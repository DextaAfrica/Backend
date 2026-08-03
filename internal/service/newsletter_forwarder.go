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

// NewsletterForwarder delivers a validated subscription to the production
// email/CRM provider configured via NEWSLETTER_WEBHOOK_URL, matching the
// integration boundary described in the frontend README. Like Revalidator,
// this is best-effort: the subscriber is already durably stored in Postgres
// by the time this runs, so a webhook outage must not turn into a user-facing
// failure — it's logged for follow-up instead.
type NewsletterForwarder interface {
	Forward(ctx context.Context, email string)
}

type httpNewsletterForwarder struct {
	client *http.Client
	url    string
	token  string
}

func NewNewsletterForwarder(cfg config.Newsletter) NewsletterForwarder {
	return &httpNewsletterForwarder{
		client: &http.Client{Timeout: cfg.WebhookTimeout},
		url:    cfg.WebhookURL,
		token:  cfg.WebhookToken,
	}
}

func (f *httpNewsletterForwarder) Forward(ctx context.Context, email string) {
	if f.url == "" {
		slog.Debug("newsletter: forwarding skipped, NEWSLETTER_WEBHOOK_URL not configured", "email", email)
		return
	}

	go func() {
		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		body, _ := json.Marshal(map[string]string{"email": email})
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, f.url, bytes.NewReader(body))
		if err != nil {
			slog.Warn("newsletter: build webhook request failed", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if f.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.token))
		}

		resp, err := f.client.Do(req)
		if err != nil {
			slog.Warn("newsletter: webhook request failed", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			slog.Warn("newsletter: webhook rejected request", "status", resp.StatusCode)
		}
	}()
}
