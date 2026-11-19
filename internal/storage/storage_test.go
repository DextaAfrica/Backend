package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/storage"
)

func TestValidateUploadRequest(t *testing.T) {
	tests := []struct {
		name        string
		folder      string
		filename    string
		contentType string
		size        int64
		resumeOnly  bool
		wantErr     bool
		expectedExt string
	}{
		{
			name:        "valid jpeg image",
			folder:      "portfolio",
			filename:    "villa.jpg",
			contentType: "image/jpeg",
			size:        2 * 1024 * 1024,
			resumeOnly:  false,
			wantErr:     false,
			expectedExt: ".jpg",
		},
		{
			name:        "valid webp image",
			folder:      "journal",
			filename:    "hero.webp",
			contentType: "image/webp",
			size:        1 * 1024 * 1024,
			resumeOnly:  false,
			wantErr:     false,
			expectedExt: ".webp",
		},
		{
			name:        "valid pdf document",
			folder:      "media",
			filename:    "brochure.pdf",
			contentType: "application/pdf",
			size:        5 * 1024 * 1024,
			resumeOnly:  false,
			wantErr:     false,
			expectedExt: ".pdf",
		},
		{
			name:        "valid candidate resume pdf",
			folder:      "resumes",
			filename:    "john_doe_cv.pdf",
			contentType: "application/pdf",
			size:        3 * 1024 * 1024,
			resumeOnly:  true,
			wantErr:     false,
			expectedExt: ".pdf",
		},
		{
			name:        "invalid resume image type",
			folder:      "resumes",
			filename:    "avatar.png",
			contentType: "image/png",
			size:        1 * 1024 * 1024,
			resumeOnly:  true,
			wantErr:     true,
		},
		{
			name:        "exceeded image size",
			folder:      "portfolio",
			filename:    "huge.png",
			contentType: "image/png",
			size:        20 * 1024 * 1024, // 20 MB > 15 MB limit
			resumeOnly:  false,
			wantErr:     true,
		},
		{
			name:        "unsupported folder",
			folder:      "system_secrets",
			filename:    "image.png",
			contentType: "image/png",
			size:        500 * 1024,
			resumeOnly:  false,
			wantErr:     true,
		},
		{
			name:        "unsupported content type executable",
			folder:      "media",
			filename:    "malware.exe",
			contentType: "application/x-msdownload",
			size:        500 * 1024,
			resumeOnly:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := storage.ValidateUploadRequest(tt.folder, tt.filename, tt.contentType, tt.size, tt.resumeOnly)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUploadRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && ext != tt.expectedExt {
				t.Errorf("ValidateUploadRequest() ext = %v, expected %v", ext, tt.expectedExt)
			}
		})
	}
}

func TestGenerateObjectKey(t *testing.T) {
	key := storage.GenerateObjectKey("portfolio", ".webp")

	parts := strings.Split(key, "/")
	if len(parts) != 4 {
		t.Fatalf("expected key to have 4 parts (<folder>/<year>/<month>/<id>.<ext>), got: %s", key)
	}

	if parts[0] != "portfolio" {
		t.Errorf("expected folder 'portfolio', got %s", parts[0])
	}

	filename := parts[3]
	if !strings.HasSuffix(filename, ".webp") {
		t.Errorf("expected suffix .webp, got %s", filename)
	}

	rawID := strings.TrimSuffix(filename, ".webp")
	if _, err := uuid.Parse(rawID); err != nil {
		t.Errorf("expected valid UUID in object key, got %s (err: %v)", rawID, err)
	}
}

func TestUnconfiguredStorageBehavior(t *testing.T) {
	cfg := config.Storage{
		Endpoint: "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com",
		Bucket:   "dextaafricastorage",
	}

	svc := storage.NewS3Service(cfg)
	if svc.IsConfigured() {
		t.Fatal("expected IsConfigured to be false when credentials are empty")
	}

	ctx := context.Background()
	_, err := svc.PresignUpload(ctx, storage.PresignUploadRequest{
		FileName:    "test.jpg",
		ContentType: "image/jpeg",
		Folder:      "media",
	})
	if err == nil {
		t.Fatal("expected error on PresignUpload when unconfigured")
	}

	publicURL := svc.GetPublicURL("portfolio/2026/11/villa.jpg")
	expected := "https://e9b3685f470b0935209bd7e768109d2a.r2.cloudflarestorage.com/dextaafricastorage/portfolio/2026/11/villa.jpg"
	if publicURL != expected {
		t.Errorf("GetPublicURL = %s, want %s", publicURL, expected)
	}
}
