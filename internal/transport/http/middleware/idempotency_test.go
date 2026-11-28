package middleware

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type mockIdempotencyRepo struct {
	mu      sync.Mutex
	records map[string]*domain.IdempotencyRecord
}

func (m *mockIdempotencyRepo) LockOrGet(ctx context.Context, key, path, method, bodyHash string, ttl time.Duration) (*domain.IdempotencyRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rec, ok := m.records[key]; ok {
		if rec.RequestBodyHash != bodyHash {
			return nil, false, errors.New("idempotency key reused with different payload")
		}
		return rec, true, nil
	}

	rec := &domain.IdempotencyRecord{
		Key:             key,
		RequestPath:     path,
		RequestMethod:   method,
		RequestBodyHash: bodyHash,
		LockedUntil:     time.Now().Add(ttl),
		CreatedAt:       time.Now(),
	}
	m.records[key] = rec
	return rec, false, nil
}

func (m *mockIdempotencyRepo) SaveResponse(ctx context.Context, key string, statusCode int, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rec, ok := m.records[key]; ok {
		rec.StatusCode = statusCode
		rec.ResponseBody = body
	}
	return nil
}

func TestIdempotencyMiddleware(t *testing.T) {
	repo := &mockIdempotencyRepo{records: make(map[string]*domain.IdempotencyRecord)}
	middleware := RequireIdempotency(repo, 1*time.Hour)

	var executionCount int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executionCount, 1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"id":"test-123"}`))
	})

	idempotentHandler := middleware(handler)

	// First request with Idempotency-Key
	req1 := httptest.NewRequest("POST", "/forms/submit", bytes.NewBufferString(`{"name":"Jane Doe"}`))
	req1.Header.Set("Idempotency-Key", "req-unique-key-1")
	rr1 := httptest.NewRecorder()
	idempotentHandler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("first request expected 201 Created, got %d", rr1.Code)
	}
	if atomic.LoadInt32(&executionCount) != 1 {
		t.Fatalf("expected handler to execute once, executed %d times", executionCount)
	}

	// Second request with SAME Idempotency-Key (simulating retry or double click)
	req2 := httptest.NewRequest("POST", "/forms/submit", bytes.NewBufferString(`{"name":"Jane Doe"}`))
	req2.Header.Set("Idempotency-Key", "req-unique-key-1")
	rr2 := httptest.NewRecorder()
	idempotentHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("second request expected 201 Created, got %d", rr2.Code)
	}
	if rr2.Header().Get("X-Cache") != "IDEMPOTENT-REPLAY" {
		t.Fatalf("expected X-Cache header IDEMPOTENT-REPLAY, got '%s'", rr2.Header().Get("X-Cache"))
	}
	// Execution count MUST remain 1!
	if atomic.LoadInt32(&executionCount) != 1 {
		t.Fatalf("handler executed again! expected 1, got %d", executionCount)
	}
}
