package uploads

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"app/internal/db"
	"app/internal/errs"

	"github.com/jackc/pgx/v5/pgtype"
)

// UploadConfig holds configuration for file uploads
type UploadConfig struct {
	UploadFolder string
	BaseURL      string
	MaxFileSize  int64
	AllowedTypes []string
	GetFolderID  func(ctx context.Context, userID int32) (int32, error)
	// Storage holds the file bytes. Nil means LocalStorage on UploadFolder and BaseURL.
	Storage Storage
}

// DefaultUploadConfig returns a default configuration
func DefaultUploadConfig(uploadFolder, baseURL string) *UploadConfig {
	return &UploadConfig{
		UploadFolder: uploadFolder,
		BaseURL:      baseURL,
		MaxFileSize:  50 * 1024 * 1024, // 50MB
		AllowedTypes: []string{
			".jpg", ".jpeg", ".png", ".gif", ".webp",
			".pdf", ".doc", ".docx", ".txt",
			".mp4", ".avi", ".mov",
			".mp3", ".wav", ".ogg",
		},
		GetFolderID: func(ctx context.Context, userID int32) (int32, error) {
			return userID, nil
		},
	}
}

// UploadService handles file upload operations
type UploadService struct {
	queries *db.Queries
	config  *UploadConfig
	storage Storage
}

// PaginatedUploads is one page of a user's uploads
type PaginatedUploads struct {
	Data  []db.Upload
	Total int64
}

// NewUploadService creates a new upload service
func NewUploadService(queries *db.Queries, config *UploadConfig) *UploadService {
	storage := config.Storage
	if storage == nil {
		storage = NewLocalStorage(config.UploadFolder, config.BaseURL)
	}
	return &UploadService{
		queries: queries,
		config:  config,
		storage: storage,
	}
}

// MaxFileSize is the largest upload the service accepts, in bytes.
func (s *UploadService) MaxFileSize() int64 {
	return s.config.MaxFileSize
}

// matchesContent compares what the file claims to be (its extension) with what its first
// bytes say it is. The extension decides the Content-Type the file is later served with,
// so an HTML page renamed to .jpg must not get in.
func matchesContent(fileType, ext, detected string) bool {
	switch {
	case fileType == "image":
		return strings.HasPrefix(detected, "image/")
	case ext == ".pdf":
		return detected == "application/pdf"
	default:
		return !strings.HasPrefix(detected, "text/html") && !strings.HasPrefix(detected, "text/xml")
	}
}

// GetFileType determines the file type based on extension
func (s *UploadService) GetFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "image"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv":
		return "video"
	case ".mp3", ".wav", ".ogg", ".aac", ".flac":
		return "audio"
	case ".pdf", ".doc", ".docx", ".txt", ".xls", ".xlsx":
		return "document"
	default:
		return "other"
	}
}

// IsValidFileType checks if the file type is allowed
func (s *UploadService) IsValidFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowedExt := range s.config.AllowedTypes {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// GenerateRandomName generates a random filename that keeps only the extension of the original
func (s *UploadService) GenerateRandomName(originalName string) (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	return fmt.Sprintf("%d_%s%s", time.Now().Unix(), hex.EncodeToString(randomBytes), ext), nil
}

// UploadFile validates a file, stores it and records it in the database
func (s *UploadService) UploadFile(ctx context.Context, file *multipart.FileHeader, userID int32) (*db.Upload, error) {
	if !s.IsValidFileType(file.Filename) {
		return nil, errs.WrapBadRequest(
			errs.ErrKeyUploadTypeNotAllowed,
			"File type not allowed",
			fmt.Errorf("file type not allowed: %s", filepath.Ext(file.Filename)),
		)
	}

	if file.Size > s.config.MaxFileSize {
		return nil, errs.WrapBadRequest(
			errs.ErrKeyUploadTooLarge,
			"File too large",
			fmt.Errorf("file too large: %d bytes (max %d bytes)", file.Size, s.config.MaxFileSize),
		).WithDetails(map[string]interface{}{"max_bytes": s.config.MaxFileSize})
	}

	if file.Size == 0 {
		return nil, errs.NewBadRequestError(errs.ErrKeyUploadEmpty, "File is empty")
	}

	src, err := file.Open()
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to open uploaded file", err)
	}
	defer src.Close()

	// The Content-Type header is whatever the client says; the first bytes are not.
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to read uploaded file", err)
	}
	head = head[:n]
	mimeType := http.DetectContentType(head)

	fileType := s.GetFileType(file.Filename)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !matchesContent(fileType, ext, mimeType) {
		return nil, errs.WrapBadRequest(
			errs.ErrKeyUploadTypeNotAllowed,
			"File content does not match its type",
			fmt.Errorf("extension %s but content is %s", ext, mimeType),
		)
	}

	folderID, err := s.config.GetFolderID(ctx, userID)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to get folder ID", err)
	}

	filename, err := s.GenerateRandomName(file.Filename)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to generate file name", err)
	}
	relativePath := path.Join(strconv.Itoa(int(folderID)), filename)

	if err := s.storage.Save(ctx, relativePath, io.MultiReader(strings.NewReader(string(head)), src)); err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to store file", err)
	}

	upload, err := s.queries.CreateUpload(ctx, db.CreateUploadParams{
		UserID:           userID,
		FolderID:         folderID,
		Type:             fileType,
		RelativePath:     relativePath,
		OriginalFilename: filepath.Base(file.Filename),
		FileSize:         file.Size,
		MimeType:         pgtype.Text{String: mimeType, Valid: true},
	})
	if err != nil {
		_ = s.storage.Delete(ctx, relativePath)
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to save upload to database", err)
	}

	return &upload, nil
}

// GetUpload retrieves an upload by ID and user ID.
// Returns ErrUploadNotFound if the upload doesn't exist or doesn't belong to the user.
// This method can be used internally by other services to retrieve upload information.
func (s *UploadService) GetUpload(ctx context.Context, uploadID, userID int32) (*db.Upload, error) {
	upload, err := s.queries.GetUploadByIDAndUserID(ctx, db.GetUploadByIDAndUserIDParams{
		ID:     uploadID,
		UserID: userID,
	})
	if err != nil {
		return nil, ErrUploadNotFound
	}
	return &upload, nil
}

// ListUploadsPaginated returns one page of a user's uploads, newest first.
func (s *UploadService) ListUploadsPaginated(ctx context.Context, userID, page, pageSize int32) (*PaginatedUploads, error) {
	total, err := s.queries.CountUploadsByUserID(ctx, userID)
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	uploads, err := s.queries.ListUploadsByUserIDPaginated(ctx, db.ListUploadsByUserIDPaginatedParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	return &PaginatedUploads{Data: uploads, Total: total}, nil
}

// DeleteUpload deletes an upload by ID and user ID.
// This method:
//   - Verifies the upload exists and belongs to the user
//   - Deletes the record from the database
//   - Removes the file from disk
//
// Returns ErrUploadNotFound if the upload doesn't exist or doesn't belong to the user.
// This method can be used internally by other services to delete uploads.
func (s *UploadService) DeleteUpload(ctx context.Context, uploadID, userID int32) error {
	upload, err := s.GetUpload(ctx, uploadID, userID)
	if err != nil {
		return err
	}

	err = s.queries.DeleteUpload(ctx, db.DeleteUploadParams{
		ID:     uploadID,
		UserID: userID,
	})
	if err != nil {
		return errs.WrapInternal(errs.ErrKeyInternalError, "failed to delete upload", err)
	}

	if err := s.storage.Delete(ctx, upload.RelativePath); err != nil {
		return errs.WrapInternal(errs.ErrKeyInternalError, "failed to delete stored file", err)
	}

	return nil
}

// GetFullURL returns the full URL for an upload
func (s *UploadService) GetFullURL(relativePath string) string {
	return s.storage.URL(relativePath)
}

var (
	ErrUploadNotFound = errs.NewNotFoundError(
		errs.ErrKeyUploadNotFound,
		"Upload not found",
	)
)
