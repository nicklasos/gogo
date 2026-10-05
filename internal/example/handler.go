package example

import (
	"app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *ExampleService
}

func NewHandler(service *ExampleService) *Handler {
	return &Handler{service: service}
}

func exampleResponse(example *db.Example) *ExampleResponse {
	return &ExampleResponse{
		ID:          example.ID,
		UserID:      example.UserID,
		Title:       example.Title,
		Description: example.Description.String,
		CreatedAt:   internal.FormatTime(example.CreatedAt),
		UpdatedAt:   internal.FormatTime(example.UpdatedAt),
	}
}

// CreateExample creates a new example
//
//	@Summary		Create example
//	@Description	Create a new example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateExampleRequest	true	"Example details"
//	@Success		200		{object}	ExampleDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/examples [post]
func (h *Handler) CreateExample(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var req CreateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	example, err := h.service.CreateExample(c.Request.Context(), userID, req.Title, req.Description)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: exampleResponse(example)})
}

// GetExample retrieves an example by ID
//
//	@Summary		Get example
//	@Description	Get an example by ID for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Example ID"
//	@Success		200	{object}	ExampleDataResponse
//	@Failure		400	{object}	errs.ErrorResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
//	@Failure		500	{object}	errs.ErrorResponse
//	@Router			/api/v1/examples/{id} [get]
func (h *Handler) GetExample(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	example, err := h.service.GetExample(c.Request.Context(), id, userID)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: exampleResponse(example)})
}

// ListExamples lists all examples for the authenticated user with pagination
//
//	@Summary		List examples (paginated)
//	@Description	Get all examples for the authenticated user with pagination
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default: 1)"					default(1)
//	@Param			page_size	query		int	false	"Page size (default: 20, min: 1, max: 100)"	default(20)
//	@Success		200			{object}	PaginatedExamplesResponse
//	@Failure		400			{object}	errs.ErrorResponse
//	@Failure		401			{object}	errs.ErrorResponse
//	@Failure		500			{object}	errs.ErrorResponse
//	@Router			/api/v1/examples [get]
func (h *Handler) ListExamples(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}
	page, ok := middleware.Page(c)
	if !ok {
		return
	}

	result, err := h.service.ListExamplesPaginated(c.Request.Context(), userID, page.Page, page.PageSize)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	items := make([]ExampleResponse, len(result.Data))
	for i := range result.Data {
		items[i] = *exampleResponse(&result.Data[i])
	}

	c.JSON(http.StatusOK, PaginatedExamplesResponse{
		Data:       items,
		Pagination: internal.NewPaginationMeta(result.Total, page.Page, page.PageSize),
	})
}

// UpdateExample updates an existing example
//
//	@Summary		Update example
//	@Description	Update an existing example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"Example ID"
//	@Param			request	body		UpdateExampleRequest	true	"Example details"
//	@Success		200		{object}	ExampleDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		404		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/examples/{id} [put]
func (h *Handler) UpdateExample(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	var req UpdateExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	example, err := h.service.UpdateExample(c.Request.Context(), id, userID, req.Title, req.Description)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, ExampleDataResponse{Data: exampleResponse(example)})
}

// DeleteExample deletes an example
//
//	@Summary		Delete example
//	@Description	Delete an example for the authenticated user
//	@Tags			examples
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Example ID"
//	@Success		200	{object}	internal.MessageResponse
//	@Failure		400	{object}	errs.ErrorResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
//	@Failure		500	{object}	errs.ErrorResponse
//	@Router			/api/v1/examples/{id} [delete]
func (h *Handler) DeleteExample(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteExample(c.Request.Context(), id, userID); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Example deleted successfully")
}
