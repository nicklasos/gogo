package uploads

import (
	"errors"
	"net/http"

	"app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

// multipartOverhead is room for the form boundaries around the file itself
const multipartOverhead = 1 << 20

type Handler struct {
	service *UploadService
}

func NewHandler(service *UploadService) *Handler {
	return &Handler{service: service}
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
		CreatedAt:        internal.FormatTime(upload.CreatedAt),
		UpdatedAt:        internal.FormatTime(upload.UpdatedAt),
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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
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
		errs.RespondWithBadRequest(c, errs.ErrKeyUploadEmpty, "No file uploaded")
		return
	}

	upload, err := h.service.UploadFile(c.Request.Context(), file, userID)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	uploadID, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	upload, err := h.service.GetUpload(c.Request.Context(), uploadID, userID)
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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	pagination, ok := middleware.Page(c)
	if !ok {
		return
	}

	result, err := h.service.ListUploadsPaginated(c.Request.Context(), userID, pagination.Page, pagination.PageSize)
	if err != nil {
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
//	@Success		200	{object}	internal.MessageResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
//	@Router			/api/v1/uploads/{id} [delete]
func (h *Handler) DeleteUpload(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	uploadID, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteUpload(c.Request.Context(), uploadID, userID); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Upload deleted successfully")
}
