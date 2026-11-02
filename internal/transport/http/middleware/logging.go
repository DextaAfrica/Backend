package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder captures the status code written by downstream handlers so
// it can be logged after the fact — http.ResponseWriter doesn't expose it.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytesWritten int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// Logging emits one structured log line per request with the fields needed
// to correlate it with an error response or an upstream trace: method, path,
// status, duration, and request ID.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"request_id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytesWritten,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
