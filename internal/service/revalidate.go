package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
)

// Revalidator notifies the Next.js frontend that content changed so it can
// invalidate its cache immediately, per the contract in the frontend's
// README: POST /api/revalidate with `Authorization: Bearer
// <CONTENT_REVALIDATION_SECRET>`. It is deliberately best-effort — a
// revalidation failure must never roll back or fail a content save that
// already succeeded in the database; the frontend's own cache TTL is the
// fallback.
type Revalidator interface {
	Trigger(ctx context.Context, tag string)
}

type httpRevalidator struct {
	client *http.Client
	url    string
	secret string
}

func NewRevalidator(cfg config.Frontend) Revalidator {
	return &httpRevalidator{
		client: &http.Client{Timeout: cfg.RevalidateTimeout},
		url:    cfg.RevalidateURL,
		secret: cfg.RevalidateSecret,
	}
}

func (r *httpRevalidator) Trigger(ctx context.Context, tag string) {
	if r.url == "" {
		slog.Debug("revalidate: skipped, FRONTEND_REVALIDATE_URL not configured", "tag", tag)
		return
	}

	go func() {
		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		body, _ := json.Marshal(map[string]string{"tag": tag})
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, r.url, bytes.NewReader(body))
		if err != nil {
			slog.Warn("revalidate: build request failed", "tag", tag, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.secret))

		resp, err := r.client.Do(req)
		if err != nil {
			slog.Warn("revalidate: request failed", "tag", tag, "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			slog.Warn("revalidate: frontend rejected request", "tag", tag, "status", resp.StatusCode)
		}
	}()
}
