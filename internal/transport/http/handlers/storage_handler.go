package handlers

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/storage"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

// StorageHandler exposes endpoints for client direct-to-cloud presigning and streaming uploads.
type StorageHandler struct {
	storage storage.Service
}

func NewStorageHandler(storageService storage.Service) *StorageHandler {
	return &StorageHandler{storage: storageService}
}

// PresignAdminUpload generates a presigned PUT URL for direct R2 upload by authenticated CMS editors.
// POST /admin/uploads/presign
func (h *StorageHandler) PresignAdminUpload(w http.ResponseWriter, r *http.Request) {
	var req request.PresignUploadRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	result, err := h.storage.PresignUpload(r.Context(), storage.PresignUploadRequest{
		FileName:    req.FileName,
		ContentType: req.ContentType,
		Folder:      req.Folder,
		SizeBytes:   req.SizeBytes,
		IsPublic:    true,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// DirectAdminUpload receives a multipart file from the admin dashboard and streams it to R2.
// POST /admin/uploads
func (h *StorageHandler) DirectAdminUpload(w http.ResponseWriter, r *http.Request) {
	// Hard-bound request body to prevent unbounded memory/disk spooling (gosec G111/G117)
	const maxUploadBytes = 60 << 20 // 60MB max request size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	// 32MB max memory buffer before spooling to temp file
	// #nosec G120 -- request body is bounded by http.MaxBytesReader above
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.Error(w, r, apperror.New(apperror.CodeBadRequest, "failed to parse multipart form: "+err.Error()))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, r, apperror.Validation("missing file", map[string]string{
			"file": "multipart field 'file' is required",
		}))
		return
	}
	defer file.Close()

	folder := r.FormValue("folder")
	if folder == "" {
		folder = storage.FolderMedia
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		if ext := filepath.Ext(header.Filename); ext != "" {
			contentType = mime.TypeByExtension(ext)
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ext, valErr := storage.ValidateUploadRequest(folder, header.Filename, contentType, header.Size, false)
	if valErr != nil {
		response.Error(w, r, valErr)
		return
	}

	key := storage.GenerateObjectKey(folder, ext)
	res, err := h.storage.Upload(r.Context(), key, file, header.Size, contentType)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	res.FileName = header.Filename
	response.JSON(w, http.StatusCreated, res)
}

// DeleteAdminUpload removes an asset from R2 storage.
// DELETE /admin/uploads
func (h *StorageHandler) DeleteAdminUpload(w http.ResponseWriter, r *http.Request) {
	var key string
	if r.Method == http.MethodDelete && r.Header.Get("Content-Type") == "application/json" {
		var req request.DeleteUploadRequest
		if err := request.Decode(w, r, &req); err == nil {
			key = req.Key
		}
	}
	if key == "" {
		key = r.URL.Query().Get("key")
	}

	if strings.TrimSpace(key) == "" {
		response.Error(w, r, apperror.Validation("missing key", map[string]string{
			"key": "query parameter or body field 'key' is required",
		}))
		return
	}

	if err := h.storage.Delete(r.Context(), key); err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"status":  "deleted",
		"key":     key,
		"message": "asset deleted successfully",
	})
}

// PresignPublicResume allows website job candidates to request a signed upload URL specifically for their CV.
// POST /uploads/resume/presign
func (h *StorageHandler) PresignPublicResume(w http.ResponseWriter, r *http.Request) {
	var req request.PresignUploadRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	// Strictly enforce candidate resume constraints
	ext, valErr := storage.ValidateUploadRequest(storage.FolderResumes, req.FileName, req.ContentType, req.SizeBytes, true)
	if valErr != nil {
		response.Error(w, r, valErr)
		return
	}

	key := storage.GenerateObjectKey(storage.FolderResumes, ext)
	uploadURL, err := h.storage.PresignUpload(r.Context(), storage.PresignUploadRequest{
		FileName:    req.FileName,
		ContentType: req.ContentType,
		Folder:      storage.FolderResumes,
		SizeBytes:   req.SizeBytes,
		IsPublic:    false,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"upload_url": uploadURL.UploadURL,
		"file_url":   uploadURL.FileURL,
		"key":        key,
		"headers":    uploadURL.Headers,
		"expires_in": uploadURL.ExpiresIn,
	})
}

// DirectPublicResume allows job applicants to upload their resume directly via multipart form.
// POST /uploads/resume
func (h *StorageHandler) DirectPublicResume(w http.ResponseWriter, r *http.Request) {
	// Hard-bound request body to prevent unbounded memory/disk spooling (gosec G111/G117)
	const maxResumeBytes = 25 << 20 // 25MB max request size
	r.Body = http.MaxBytesReader(w, r.Body, maxResumeBytes)

	// #nosec G120 -- request body is bounded by http.MaxBytesReader above
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		response.Error(w, r, apperror.New(apperror.CodeBadRequest, "failed to parse multipart form: "+err.Error()))
		return
	}

	file, header, err := r.FormFile("resume")
	if err != nil {
		// Fallback to "file" field name
		var fallbackErr error
		file, header, fallbackErr = r.FormFile("file")
		if fallbackErr != nil {
			response.Error(w, r, apperror.Validation("missing resume file", map[string]string{
				"resume": "multipart field 'resume' or 'file' is required",
			}))
			return
		}
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		if ext := filepath.Ext(header.Filename); ext != "" {
			contentType = mime.TypeByExtension(ext)
		}
	}

	ext, valErr := storage.ValidateUploadRequest(storage.FolderResumes, header.Filename, contentType, header.Size, true)
	if valErr != nil {
		response.Error(w, r, valErr)
		return
	}

	key := storage.GenerateObjectKey(storage.FolderResumes, ext)
	res, err := h.storage.Upload(r.Context(), key, file, header.Size, contentType)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"resume_url": res.FileURL,
		"key":        res.Key,
		"file_name":  header.Filename,
		"size":       header.Size,
	})
}

// PresignDownloadPrivate generates a temporary signed GET URL for downloading private applicant resumes.
// GET /admin/uploads/private?key=...
func (h *StorageHandler) PresignDownloadPrivate(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if strings.TrimSpace(key) == "" {
		response.Error(w, r, apperror.Validation("missing key", map[string]string{
			"key": "query parameter 'key' is required",
		}))
		return
	}

	url, err := h.storage.PresignDownload(r.Context(), key, 15*time.Minute)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"download_url": url,
		"key":          key,
		"expires_in":   900,
	})
}
