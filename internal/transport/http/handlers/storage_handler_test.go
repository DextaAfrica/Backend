package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DextaAfrica/Backend/internal/storage"
	"github.com/DextaAfrica/Backend/internal/transport/http/handlers"
)

type mockStorage struct {
	presignUploadFn   func(ctx context.Context, req storage.PresignUploadRequest) (*storage.PresignedUpload, error)
	uploadFn          func(ctx context.Context, key string, body io.Reader, size int64, contentType string) (*storage.UploadResult, error)
	deleteFn          func(ctx context.Context, key string) error
	presignDownloadFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	getPublicURLFn    func(key string) string
	isConfiguredFn    func() bool
}

func (m *mockStorage) PresignUpload(ctx context.Context, req storage.PresignUploadRequest) (*storage.PresignedUpload, error) {
	if m.presignUploadFn != nil {
		return m.presignUploadFn(ctx, req)
	}
	return &storage.PresignedUpload{
		UploadURL: "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/upload-url",
		FileURL:   "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/" + req.Folder + "/test.webp",
		Key:       req.Folder + "/test.webp",
		Headers:   map[string]string{"Content-Type": req.ContentType},
		ExpiresIn: 900,
	}, nil
}

func (m *mockStorage) Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, body, size, contentType)
	}
	return &storage.UploadResult{
		FileURL:     "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/" + key,
		Key:         key,
		Size:        size,
		ContentType: contentType,
	}, nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockStorage) PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if m.presignDownloadFn != nil {
		return m.presignDownloadFn(ctx, key, ttl)
	}
	return "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/" + key + "?signed=true", nil
}

func (m *mockStorage) GetPublicURL(key string) string {
	if m.getPublicURLFn != nil {
		return m.getPublicURLFn(key)
	}
	return "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/" + key
}

func (m *mockStorage) IsConfigured() bool {
	if m.isConfiguredFn != nil {
		return m.isConfiguredFn()
	}
	return true
}

func TestPresignAdminUpload(t *testing.T) {
	mock := &mockStorage{}
	h := handlers.NewStorageHandler(mock)

	t.Run("successful presign", func(t *testing.T) {
		body := map[string]any{
			"fileName":    "villa.webp",
			"contentType": "image/webp",
			"folder":      "portfolio",
			"sizeBytes":   1024 * 500,
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/admin/uploads/presign", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.PresignAdminUpload(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["upload_url"] == nil || resp["file_url"] == nil || resp["key"] == nil {
			t.Errorf("missing expected fields in response: %v", resp)
		}
	})

	t.Run("missing file name validation failure", func(t *testing.T) {
		body := map[string]any{
			"contentType": "image/webp",
			"folder":      "portfolio",
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/admin/uploads/presign", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.PresignAdminUpload(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for missing filename, got %d", rr.Code)
		}
	})

	t.Run("invalid folder validation failure", func(t *testing.T) {
		body := map[string]any{
			"fileName":    "villa.webp",
			"contentType": "image/webp",
			"folder":      "malicious_folder",
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/admin/uploads/presign", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.PresignAdminUpload(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for invalid folder, got %d", rr.Code)
		}
	})
}

func TestDeleteAdminUpload(t *testing.T) {
	mock := &mockStorage{}
	h := handlers.NewStorageHandler(mock)

	t.Run("successful deletion via query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/uploads?key=portfolio/2026/11/villa.webp", nil)
		rr := httptest.NewRecorder()

		h.DeleteAdminUpload(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing key validation failure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/uploads", nil)
		rr := httptest.NewRecorder()

		h.DeleteAdminUpload(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for missing key, got %d", rr.Code)
		}
	})
}

func TestPresignPublicResume(t *testing.T) {
	mock := &mockStorage{}
	h := handlers.NewStorageHandler(mock)

	t.Run("successful resume presign", func(t *testing.T) {
		body := map[string]any{
			"fileName":    "resume.pdf",
			"contentType": "application/pdf",
			"sizeBytes":   2 * 1024 * 1024,
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/uploads/resume/presign", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.PresignPublicResume(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("reject non-document for resume", func(t *testing.T) {
		body := map[string]any{
			"fileName":    "avatar.png",
			"contentType": "image/png",
			"sizeBytes":   500 * 1024,
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/uploads/resume/presign", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.PresignPublicResume(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for non-resume MIME type, got %d", rr.Code)
		}
	})
}

func TestPresignDownloadPrivate(t *testing.T) {
	mock := &mockStorage{}
	h := handlers.NewStorageHandler(mock)

	t.Run("successful download URL generation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/uploads/private?key=resumes/2026/11/candidate.pdf", nil)
		rr := httptest.NewRecorder()

		h.PresignDownloadPrivate(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["download_url"] == nil {
			t.Fatalf("expected download_url in response")
		}
	})

	t.Run("redirect parameter directs browser directly to signed download URL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/uploads/private?key=resumes/2026/11/candidate.pdf&redirect=true", nil)
		rr := httptest.NewRecorder()

		h.PresignDownloadPrivate(rr, req)

		if rr.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected status 307, got %d", rr.Code)
		}
	})
}
