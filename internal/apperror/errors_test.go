package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPStatus(t *testing.T) {
	cases := []struct {
		code Code
		want int
	}{
		{CodeValidation, http.StatusBadRequest},
		{CodeBadRequest, http.StatusBadRequest},
		{CodeUnauthorized, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeNotFound, http.StatusNotFound},
		{CodeConflict, http.StatusConflict},
		{CodeRateLimited, http.StatusTooManyRequests},
		{CodeUnavailable, http.StatusServiceUnavailable},
		{CodeInternal, http.StatusInternalServerError},
		{Code("unknown"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		err := New(tc.code, "message")
		if got := err.HTTPStatus(); got != tc.want {
			t.Errorf("Code %q: HTTPStatus() = %d, want %d", tc.code, got, tc.want)
		}
	}
}

func TestAs_WrapsUnknownErrorsAsInternal(t *testing.T) {
	plain := errors.New("boom")
	got := As(plain)

	if got.Code != CodeInternal {
		t.Fatalf("As(plain error) code = %q, want %q", got.Code, CodeInternal)
	}
	if !errors.Is(got.Unwrap(), plain) {
		t.Fatalf("As(plain error) should wrap the original error")
	}
}

func TestAs_PassesThroughExistingAppError(t *testing.T) {
	original := NotFound("widget")
	got := As(original)

	if got != original {
		t.Fatalf("As(*Error) should return the same instance, got a different pointer")
	}
}

func TestAs_NilIsNil(t *testing.T) {
	if As(nil) != nil {
		t.Fatalf("As(nil) should return nil")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("db exploded")
	err := Wrap(CodeInternal, "failed", cause)

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is should find the wrapped cause")
	}
}
