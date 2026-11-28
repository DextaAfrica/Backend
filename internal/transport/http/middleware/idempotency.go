package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type responseCaptureWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *responseCaptureWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCaptureWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func RequireIdempotency(repo domain.IdempotencyRepository, ttl time.Duration) func(http.Handler) http.Handler {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Idempotency applies to mutating operations (POST, PUT, PATCH)
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				key = r.Header.Get("X-Idempotency-Key")
			}
			// If no idempotency key is supplied, proceed without replay cache
			if key == "" || repo == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Read and buffer request body for hashing
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
			bodyHashBytes := sha256.Sum256(bodyBytes)
			bodyHash := hex.EncodeToString(bodyHashBytes[:])

			record, exists, err := repo.LockOrGet(r.Context(), key, r.URL.Path, r.Method, bodyHash, ttl)
			if err != nil {
				response.Error(w, r, apperror.New(apperror.CodeConflict, err.Error()))
				return
			}

			if exists {
				// If status code is already recorded, replay cached response
				if record.StatusCode > 0 {
					w.Header().Set("X-Cache", "IDEMPOTENT-REPLAY")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(record.StatusCode)
					_, _ = w.Write(record.ResponseBody) // #nosec G705 -- replaying verified API response payload
					return
				}

				// If no status code and lock is still active, another request is currently processing
				if record.LockedUntil.After(time.Now().UTC()) {
					response.Error(w, r, apperror.New(apperror.CodeConflict, "concurrent request in progress with this idempotency key"))
					return
				}
			}

			// Capture response and save on completion
			recorder := &responseCaptureWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r)

			if recorder.statusCode > 0 && recorder.statusCode < 500 {
				_ = repo.SaveResponse(r.Context(), key, recorder.statusCode, recorder.body.Bytes())
			}
		})
	}
}
