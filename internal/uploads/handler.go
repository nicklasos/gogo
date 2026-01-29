package uploads

import (
	"net/http"
	"strconv"

	"app/internal/errs"
	"app/internal/logger"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *UploadService
	logger  *logger.Logger
}

func NewHandler(service *UploadService, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// UploadFile uploads a file
//
//	@Summary		Upload file
//	@Description	Upload a file for the authenticated user
//	@Tags			uploads
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		Bearer
//	@Param			file	formData	file				true	"File to upload"
//	@Success		200		{object}	UploadDataResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/v1/uploads [post]
func (h *Handler) UploadFile(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get uploaded file", "error", err)
		errs.RespondWithBadRequest(c, errs.ErrKeyValidationError, "No file uploaded")
		return
	}

	upload, err := h.service.UploadFile(c.Request.Context(), file, userID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to upload file", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	h.logger.InfoContext(c.Request.Context(), "File uploaded successfully", "upload_id", upload.ID, "user_id", userID)

	c.JSON(http.StatusOK, UploadDataResponse{
		Data: &UploadResponse{
			ID:               upload.ID,
			UserID:           upload.UserID,
			FolderID:         upload.FolderID,
			Type:             upload.Type,
			RelativePath:     upload.RelativePath,
			FullURL:          h.service.GetFullURL(upload.RelativePath),
			OriginalFilename: upload.OriginalFilename,
			FileSize:         upload.FileSize,
			MimeType:         upload.MimeType.String,
			CreatedAt:        upload.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        upload.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

// GetUpload retrieves an upload by ID
//
//	@Summary		Get upload
//	@Description	Get an upload by ID
//	@Tags			uploads
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"Upload ID"
//	@Success		200	{object}	UploadDataResponse
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Router			/api/v1/uploads/{id} [get]
func (h *Handler) GetUpload(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	uploadIDStr := c.Param("id")
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 32)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyValidationError, "Invalid upload ID")
		return
	}

	upload, err := h.service.GetUpload(c.Request.Context(), int32(uploadID), userID)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UploadDataResponse{
		Data: &UploadResponse{
			ID:               upload.ID,
			UserID:           upload.UserID,
			FolderID:         upload.FolderID,
			Type:             upload.Type,
			RelativePath:     upload.RelativePath,
			FullURL:          h.service.GetFullURL(upload.RelativePath),
			OriginalFilename: upload.OriginalFilename,
			FileSize:         upload.FileSize,
			MimeType:         upload.MimeType.String,
			CreatedAt:        upload.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        upload.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

// ListUploads lists all uploads for the authenticated user
//
//	@Summary		List uploads
//	@Description	List all uploads for the authenticated user
//	@Tags			uploads
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	UploadsListResponse
//	@Failure		401	{object}	map[string]interface{}
//	@Router			/api/v1/uploads [get]
func (h *Handler) ListUploads(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	uploads, err := h.service.ListUploads(c.Request.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list uploads", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	response := make([]UploadResponse, len(uploads))
	for i, upload := range uploads {
		response[i] = UploadResponse{
			ID:               upload.ID,
			UserID:           upload.UserID,
			FolderID:         upload.FolderID,
			Type:             upload.Type,
			RelativePath:     upload.RelativePath,
			FullURL:          h.service.GetFullURL(upload.RelativePath),
			OriginalFilename: upload.OriginalFilename,
			FileSize:         upload.FileSize,
			MimeType:         upload.MimeType.String,
			CreatedAt:        upload.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        upload.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, UploadsListResponse{
		Data: response,
	})
}

// DeleteUpload deletes an upload
//
//	@Summary		Delete upload
//	@Description	Delete an upload by ID
//	@Tags			uploads
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"Upload ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Router			/api/v1/uploads/{id} [delete]
func (h *Handler) DeleteUpload(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	uploadIDStr := c.Param("id")
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 32)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyValidationError, "Invalid upload ID")
		return
	}

	err = h.service.DeleteUpload(c.Request.Context(), int32(uploadID), userID)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Data: struct {
			Message string `json:"message"`
		}{
			Message: "Upload deleted successfully",
		},
	})
}
