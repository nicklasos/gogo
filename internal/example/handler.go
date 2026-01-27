package example

import (
	"app/internal"
	"app/internal/errs"
	"app/internal/logger"
	"app/internal/middleware"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *ExampleService
	logger  *logger.Logger
}

func NewHandler(service *ExampleService, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// CreateExample creates a new example
//
//	@Summary		Create example
//	@Description	Create a new example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		CreateExampleRequest	true	"Example details"
//	@Success		200		{object}	ExampleDataResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/examples [post]
func (h *Handler) CreateExample(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	// Validation error handling example: Use RespondWithValidationError for binding errors
	var req CreateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	// Service error handling example: Use RespondWithError for domain errors
	example, err := h.service.CreateExample(c.Request.Context(), userID, req.Title, req.Description)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to create example", "error", err, "user_id", userID)
		errs.RespondWithError(c, err) // Automatically formats domain error
		return
	}

	response := ExampleResponse{
		ID:          example.ID,
		UserID:      example.UserID,
		Title:       example.Title,
		Description: example.Description.String,
		CreatedAt:   example.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   example.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: &response})
}

// GetExample retrieves an example by ID
//
//	@Summary		Get example
//	@Description	Get an example by ID for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"Example ID"
//	@Success		200	{object}	ExampleDataResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/examples/{id} [get]
func (h *Handler) GetExample(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid example ID")
		return
	}

	// Domain error handling example: Service returns domain error, handler just passes it through
	example, err := h.service.GetExample(c.Request.Context(), int32(id), userID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get example", "error", err, "example_id", id, "user_id", userID)
		errs.RespondWithError(c, err) // Domain error automatically formatted
		return
	}

	response := ExampleResponse{
		ID:          example.ID,
		UserID:      example.UserID,
		Title:       example.Title,
		Description: example.Description.String,
		CreatedAt:   example.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   example.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: &response})
}

// ListExamples lists all examples for the authenticated user with pagination
//
//	@Summary		List examples (paginated)
//	@Description	Get all examples for the authenticated user with pagination
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int		false	"Page number (default: 1)"					default(1)
//	@Param			page_size	query		int		false	"Page size (default: 20, min: 1, max: 100)"	default(20)
//	@Success		200			{object}	PaginatedExamplesResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/api/v1/examples [get]
func (h *Handler) ListExamples(c *gin.Context) {
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

	result, err := h.service.ListExamplesPaginated(c.Request.Context(), userID, pagination.Page, pagination.PageSize)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to list examples", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	// Convert db.Example to ExampleResponse
	examples := make([]ExampleResponse, len(result.Data))
	for i, ex := range result.Data {
		examples[i] = ExampleResponse{
			ID:          ex.ID,
			UserID:      ex.UserID,
			Title:       ex.Title,
			Description: ex.Description.String,
			CreatedAt:   ex.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   ex.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	response := PaginatedExamplesResponse{
		Data:       examples,
		Pagination: internal.NewPaginationMeta(result.Total, result.Page, result.PageSize),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateExample updates an existing example
//
//	@Summary		Update example
//	@Description	Update an existing example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int					true	"Example ID"
//	@Param			request	body		UpdateExampleRequest	true	"Example details"
//	@Success		200		{object}	ExampleDataResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/examples/{id} [put]
func (h *Handler) UpdateExample(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid example ID")
		return
	}

	var req UpdateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	example, err := h.service.UpdateExample(c.Request.Context(), int32(id), userID, req.Title, req.Description)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update example", "error", err, "example_id", id, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	response := ExampleResponse{
		ID:          example.ID,
		UserID:      example.UserID,
		Title:       example.Title,
		Description: example.Description.String,
		CreatedAt:   example.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   example.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: &response})
}

// DeleteExample deletes an example
//
//	@Summary		Delete example
//	@Description	Delete an example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int	true	"Example ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/examples/{id} [delete]
func (h *Handler) DeleteExample(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid example ID")
		return
	}

	err = h.service.DeleteExample(c.Request.Context(), int32(id), userID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to delete example", "error", err, "example_id", id, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	var response MessageResponse
	response.Data.Message = "Example deleted successfully"
	c.JSON(http.StatusOK, response)
}
