package uploads

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
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
}

// NewUploadService creates a new upload service
func NewUploadService(queries *db.Queries, config *UploadConfig) *UploadService {
	return &UploadService{
		queries: queries,
		config:  config,
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

// GenerateRandomName generates a random filename
func (s *UploadService) GenerateRandomName(originalName string) string {
	ext := filepath.Ext(originalName)
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	randomString := hex.EncodeToString(randomBytes)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s%s", timestamp, randomString, ext)
	return filename
}

// UploadFile uploads a file and stores it in the database
func (s *UploadService) UploadFile(ctx context.Context, file *multipart.FileHeader, userID int32) (*db.Upload, error) {
	if !s.IsValidFileType(file.Filename) {
		return nil, errs.WrapBadRequest(
			errs.ErrKeyValidationError,
			"File type not allowed",
			fmt.Errorf("file type not allowed: %s", filepath.Ext(file.Filename)),
		)
	}

	if file.Size > s.config.MaxFileSize {
		return nil, errs.WrapBadRequest(
			errs.ErrKeyValidationError,
			"File too large",
			fmt.Errorf("file too large: %d bytes (max %d bytes)", file.Size, s.config.MaxFileSize),
		)
	}

	if file.Size == 0 {
		return nil, errs.NewBadRequestError(
			errs.ErrKeyValidationError,
			"File is empty",
		)
	}

	folderID, err := s.config.GetFolderID(ctx, userID)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to get folder ID", err)
	}

	filename := s.GenerateRandomName(file.Filename)
	fileType := s.GetFileType(file.Filename)

	folderDir := strconv.Itoa(int(folderID))
	userPath := filepath.Join(s.config.UploadFolder, folderDir)

	if err := os.MkdirAll(userPath, 0755); err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to create directory", err)
	}

	filePath := filepath.Join(userPath, filename)
	relativePath := filepath.Join(folderDir, filename)

	src, err := file.Open()
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to open uploaded file", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to create destination file", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to copy file", err)
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	upload, err := s.queries.CreateUpload(ctx, db.CreateUploadParams{
		UserID:           userID,
		FolderID:         folderID,
		Type:             fileType,
		RelativePath:     relativePath,
		OriginalFilename: file.Filename,
		FileSize:         file.Size,
		MimeType:         pgtype.Text{String: mimeType, Valid: true},
	})
	if err != nil {
		os.Remove(filePath)
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

// ListUploads lists all uploads for a user.
// Returns an empty slice if the user has no uploads.
// This method can be used internally by other services to retrieve all uploads for a user.
func (s *UploadService) ListUploads(ctx context.Context, userID int32) ([]db.Upload, error) {
	uploads, err := s.queries.ListUploadsByUserID(ctx, userID)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to list uploads", err)
	}
	return uploads, nil
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

	filePath := filepath.Join(s.config.UploadFolder, upload.RelativePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return errs.WrapInternal(errs.ErrKeyInternalError, "failed to delete file from disk", err)
	}

	return nil
}

// GetFullURL returns the full URL for an upload
func (s *UploadService) GetFullURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", s.config.BaseURL, relativePath)
}

var (
	ErrUploadNotFound = errs.NewNotFoundError(
		errs.ErrKeyUploadNotFound,
		"Upload not found",
	)
)
