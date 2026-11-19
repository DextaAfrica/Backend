package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/config"
)

// S3Service implements Service using AWS SDK v2 configured for Cloudflare R2.
type S3Service struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	publicBaseURL string
	configured    bool
}

// NewS3Service constructs an enterprise storage service backed by Cloudflare R2 / S3.
func NewS3Service(cfg config.Storage) *S3Service {
	if !cfg.IsConfigured() {
		slog.Warn("storage credentials are not fully configured; storage operations will fail gracefully",
			"endpoint", cfg.Endpoint,
			"bucket", cfg.Bucket,
			"configured", false,
		)
		return &S3Service{
			bucket:        cfg.Bucket,
			publicBaseURL: cfg.BasePublicURL(),
			configured:    false,
		}
	}

	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	region := cfg.Region
	if region == "" {
		region = "auto"
	}

	client := s3.New(s3.Options{
		Region:           region,
		EndpointResolver: s3.EndpointResolverFromURL(endpoint),
		Credentials:      credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle:     true, // Recommended for Cloudflare R2 custom endpoint addressing
	})

	presignClient := s3.NewPresignClient(client)

	slog.Info("cloud storage service initialized",
		"provider", "Cloudflare R2 / S3",
		"bucket", cfg.Bucket,
		"endpoint", endpoint,
		"region", region,
		"public_url", cfg.BasePublicURL(),
	)

	return &S3Service{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		publicBaseURL: cfg.BasePublicURL(),
		configured:    true,
	}
}

func (s *S3Service) IsConfigured() bool {
	return s.configured && s.client != nil
}

// PresignUpload generates a signed PUT URL allowing the client (frontend or admin)
// to upload directly to Cloudflare R2 with strict Content-Type and expiration.
func (s *S3Service) PresignUpload(ctx context.Context, req PresignUploadRequest) (*PresignedUpload, error) {
	if !s.IsConfigured() {
		return nil, apperror.New(apperror.CodeUnavailable, "storage service is not configured with credentials")
	}

	ext, err := ValidateUploadRequest(req.Folder, req.FileName, req.ContentType, req.SizeBytes, false)
	if err != nil {
		return nil, err
	}

	key := GenerateObjectKey(req.Folder, ext)
	ttl := DefaultTTL

	presignedReq, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(req.ContentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("storage: presign put object: %w", err)
	}

	return &PresignedUpload{
		UploadURL: presignedReq.URL,
		FileURL:   s.GetPublicURL(key),
		Key:       key,
		Headers: map[string]string{
			"Content-Type": req.ContentType,
		},
		ExpiresIn: int(ttl.Seconds()),
	}, nil
}

// Upload streams a file from the backend directly into the bucket.
func (s *S3Service) Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) (*UploadResult, error) {
	if !s.IsConfigured() {
		return nil, apperror.New(apperror.CodeUnavailable, "storage service is not configured with credentials")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		input.ContentLength = size
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("storage: upload object '%s': %w", key, err)
	}

	return &UploadResult{
		FileURL:     s.GetPublicURL(key),
		Key:         key,
		Size:        size,
		ContentType: contentType,
	}, nil
}

// Delete permanently removes an object from the bucket.
func (s *S3Service) Delete(ctx context.Context, key string) error {
	if !s.IsConfigured() {
		return apperror.New(apperror.CodeUnavailable, "storage service is not configured with credentials")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return apperror.Validation("invalid key", map[string]string{"key": "object key is required"})
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: delete object '%s': %w", key, err)
	}

	return nil
}

// PresignDownload generates a temporary signed GET URL for downloading private assets.
func (s *S3Service) PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if !s.IsConfigured() {
		return "", apperror.New(apperror.CodeUnavailable, "storage service is not configured with credentials")
	}

	if ttl <= 0 {
		ttl = DefaultTTL
	}

	presignedReq, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign get object '%s': %w", key, err)
	}

	return presignedReq.URL, nil
}

// GetPublicURL returns the canonical public CDN or bucket endpoint URL for an asset.
func (s *S3Service) GetPublicURL(key string) string {
	key = strings.TrimLeft(key, "/")
	return fmt.Sprintf("%s/%s", strings.TrimRight(s.publicBaseURL, "/"), key)
}
