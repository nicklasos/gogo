package uploads

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/logger"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

// multipartOverhead is room for the form boundaries around the file itself
const multipartOverhead = 1 << 20

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

func (h *Handler) uploadResponse(upload *db.Upload) *UploadResponse {
	return &UploadResponse{
		ID:               upload.ID,
		UserID:           upload.UserID,
		FolderID:         upload.FolderID,
		Type:             upload.Type,
		RelativePath:     upload.RelativePath,
		FullURL:          h.service.GetFullURL(upload.RelativePath),
		OriginalFilename: upload.OriginalFilename,
		FileSize:         upload.FileSize,
		MimeType:         upload.MimeType.String,
		CreatedAt:        upload.CreatedAt.Time.Format(timeFormat),
		UpdatedAt:        upload.UpdatedAt.Time.Format(timeFormat),
	}
}

// UploadFile uploads a file
//
//	@Summary		Upload file
//	@Description	Upload a file for the authenticated user
//	@Tags			uploads
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file				true	"File to upload"
//	@Success		200		{object}	UploadDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/uploads [post]
func (h *Handler) UploadFile(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	// Stops an oversized body while it is still arriving, not after it has been buffered
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.service.MaxFileSize()+multipartOverhead)

	file, err := c.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			errs.RespondWithError(c, errs.NewBadRequestError(errs.ErrKeyUploadTooLarge, "File too large").
				WithDetails(map[string]interface{}{"max_bytes": h.service.MaxFileSize()}))
			return
		}
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
		Data: h.uploadResponse(upload),
	})
}

// GetUpload retrieves an upload by ID
//
//	@Summary		Get upload
//	@Description	Get an upload by ID
//	@Tags			uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Upload ID"
//	@Success		200	{object}	UploadDataResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
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
		Data: h.uploadResponse(upload),
	})
}

// ListUploads lists the authenticated user's uploads, newest first
//
//	@Summary		List uploads
//	@Description	Paginated list of the authenticated user's uploads, newest first
//	@Tags			uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number"	default(1)
//	@Param			page_size	query		int	false	"Page size"		default(20)
//	@Success		200			{object}	PaginatedUploadsResponse
//	@Failure		400			{object}	errs.ErrorResponse
//	@Failure		401			{object}	errs.ErrorResponse
//	@Router			/api/v1/uploads [get]
func (h *Handler) ListUploads(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	pagination, err := middleware.GetPaginationParamsFromContext(c, 20, 1, 100)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, err.Error())
		return
	}

	result, err := h.service.ListUploadsPaginated(c.Request.Context(), userID, pagination.Page, pagination.PageSize)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list uploads", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	items := make([]UploadResponse, len(result.Data))
	for i := range result.Data {
		items[i] = *h.uploadResponse(&result.Data[i])
	}

	c.JSON(http.StatusOK, PaginatedUploadsResponse{
		Data:       items,
		Pagination: internal.NewPaginationMeta(result.Total, pagination.Page, pagination.PageSize),
	})
}

// DeleteUpload deletes an upload
//
//	@Summary		Delete upload
//	@Description	Delete an upload by ID
//	@Tags			uploads
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Upload ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
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
