// Package storage manages asset uploading, presigning, and lifecycle operations
// backed by Cloudflare R2 / S3-compatible object storage.
package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/DextaAfrica/Backend/internal/apperror"
)

// Supported folders for organizing assets in the storage bucket.
const (
	FolderPortfolio = "portfolio"
	FolderJournal   = "journal"
	FolderPages     = "pages"
	FolderForms     = "forms"
	FolderResumes   = "resumes"
	FolderMedia     = "media"
)

// Max size constraints per asset category (in bytes).
const (
	MaxImageSize    int64 = 15 * 1024 * 1024 // 15 MB
	MaxDocumentSize int64 = 25 * 1024 * 1024 // 25 MB
	MaxVideoSize    int64 = 50 * 1024 * 1024 // 50 MB
	DefaultTTL            = 15 * time.Minute
)

// Allowed MIME types mapped to canonical file extensions.
var allowedMIMETypes = map[string]string{
	// Images
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
	"image/svg+xml": ".svg",
	"image/avif":    ".avif",
	// Documents
	"application/pdf":    ".pdf",
	"application/msword": ".doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	// Video
	"video/mp4":       ".mp4",
	"video/webm":      ".webm",
	"video/quicktime": ".mov",
}

// Resume-specific allowed MIME types (strict policy for job candidates).
var allowedResumeMIMEs = map[string]string{
	"application/pdf":    ".pdf",
	"application/msword": ".doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
}

// Service defines high-level object storage operations.
type Service interface {
	// PresignUpload generates a signed PUT URL so clients upload directly to R2.
	PresignUpload(ctx context.Context, req PresignUploadRequest) (*PresignedUpload, error)
	// Upload streams a file directly from backend into R2.
	Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) (*UploadResult, error)
	// Delete removes an asset from R2.
	Delete(ctx context.Context, key string) error
	// PresignDownload creates a temporary signed GET URL for private objects.
	PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error)
	// GetPublicURL returns the permanent public CDN/access URL for a given key.
	GetPublicURL(key string) string
	// IsConfigured returns true if credentials and bucket are ready.
	IsConfigured() bool
}

// PresignUploadRequest specifies requirements for generating a direct upload URL.
type PresignUploadRequest struct {
	FileName    string
	ContentType string
	Folder      string
	SizeBytes   int64
	IsPublic    bool
}

// PresignedUpload encapsulates the presigned URL and metadata for the client.
type PresignedUpload struct {
	UploadURL string            `json:"upload_url"`
	FileURL   string            `json:"file_url"`
	Key       string            `json:"key"`
	Headers   map[string]string `json:"headers"`
	ExpiresIn int               `json:"expires_in"` // in seconds
}

// UploadResult details an asset stored via direct backend streaming.
type UploadResult struct {
	FileURL     string `json:"file_url"`
	Key         string `json:"key"`
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// ValidateUploadRequest performs enterprise sanity checks on file metadata.
func ValidateUploadRequest(folder, filename, contentType string, size int64, resumeOnly bool) (string, error) {
	folder = strings.ToLower(strings.TrimSpace(folder))
	if folder == "" {
		folder = FolderMedia
	}

	validFolders := map[string]bool{
		FolderPortfolio: true,
		FolderJournal:   true,
		FolderPages:     true,
		FolderForms:     true,
		FolderResumes:   true,
		FolderMedia:     true,
	}

	if resumeOnly {
		folder = FolderResumes
	} else if !validFolders[folder] {
		return "", apperror.Validation("invalid upload parameters", map[string]string{
			"folder": fmt.Sprintf("invalid folder '%s'; must be one of: portfolio, journal, pages, forms, resumes, media", folder),
		})
	}

	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	var ext string
	var ok bool
	if resumeOnly || folder == FolderResumes {
		ext, ok = allowedResumeMIMEs[contentType]
		if !ok {
			return "", apperror.Validation("invalid resume format", map[string]string{
				"content_type": "invalid resume type; only PDF and Word documents (.pdf, .doc, .docx) are allowed",
			})
		}
		if size > MaxDocumentSize {
			return "", apperror.Validation("file size exceeded", map[string]string{
				"size": fmt.Sprintf("resume size exceeds maximum limit of %d MB", MaxDocumentSize/(1024*1024)),
			})
		}
	} else {
		ext, ok = allowedMIMETypes[contentType]
		if !ok {
			return "", apperror.Validation("unsupported file format", map[string]string{
				"content_type": fmt.Sprintf("unsupported content type '%s'", contentType),
			})
		}

		switch {
		case strings.HasPrefix(contentType, "image/"):
			if size > MaxImageSize {
				return "", apperror.Validation("file size exceeded", map[string]string{
					"size": fmt.Sprintf("image size exceeds maximum limit of %d MB", MaxImageSize/(1024*1024)),
				})
			}
		case strings.HasPrefix(contentType, "video/"):
			if size > MaxVideoSize {
				return "", apperror.Validation("file size exceeded", map[string]string{
					"size": fmt.Sprintf("video size exceeds maximum limit of %d MB", MaxVideoSize/(1024*1024)),
				})
			}
		default:
			if size > MaxDocumentSize {
				return "", apperror.Validation("file size exceeded", map[string]string{
					"size": fmt.Sprintf("document size exceeds maximum limit of %d MB", MaxDocumentSize/(1024*1024)),
				})
			}
		}
	}

	// Clean extension fallback if known from original filename
	origExt := strings.ToLower(filepath.Ext(filename))
	if origExt != "" {
		if mimeFromExt := mime.TypeByExtension(origExt); mimeFromExt != "" {
			if strings.HasPrefix(mimeFromExt, contentType) {
				ext = origExt
			}
		}
	}

	return ext, nil
}

// GenerateObjectKey produces a collision-free, content-partitioned path.
// Example: portfolio/2026/11/4a87c10b-8d19-4876-805c-3f4874495537.webp
func GenerateObjectKey(folder, ext string) string {
	now := time.Now().UTC()
	id := uuid.New().String()
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") && ext != "" {
		ext = "." + ext
	}
	return fmt.Sprintf("%s/%s/%s/%s%s", folder, now.Format("2006"), now.Format("01"), id, ext)
}
