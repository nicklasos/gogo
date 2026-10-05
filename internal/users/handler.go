package users

import (
	"net/http"
	"slices"

	"app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *UserService
}

func NewHandler(service *UserService) *Handler {
	return &Handler{service: service}
}

func userResponseFromDB(user db.User) UserResponse {
	roles := user.Roles
	if roles == nil {
		roles = []string{}
	}
	return UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Roles:         roles,
		EmailVerified: user.EmailVerifiedAt.Valid,
		CreatedAt:     internal.FormatTime(user.CreatedAt),
		UpdatedAt:     internal.FormatTime(user.UpdatedAt),
	}
}

// ListUsers lists users that have the given role
//
//	@Summary		List users by role
//	@Description	Paginated list of users with the given role. Admins may only list role "user"; super admins may list any role.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			role		query		string	false	"Role to list"	Enums(super-admin, admin, user)	default(user)
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size"		default(20)
//	@Success		200			{object}	PaginatedUsersResponse
//	@Failure		400			{object}	errs.ErrorResponse
//	@Failure		401			{object}	errs.ErrorResponse
//	@Failure		403			{object}	errs.ErrorResponse
//	@Failure		500			{object}	errs.ErrorResponse
//	@Router			/api/v1/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	role := c.DefaultQuery("role", middleware.RoleUser)
	if !slices.Contains(middleware.Roles, role) {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid role")
		return
	}

	pagination, ok := middleware.Page(c)
	if !ok {
		return
	}

	result, err := h.service.ListByRole(c.Request.Context(), middleware.GetUserRolesFromContext(c), role, pagination.Page, pagination.PageSize)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	items := make([]UserResponse, len(result.Data))
	for i, user := range result.Data {
		items[i] = userResponseFromDB(user)
	}

	c.JSON(http.StatusOK, PaginatedUsersResponse{
		Data:       items,
		Pagination: internal.NewPaginationMeta(result.Total, pagination.Page, pagination.PageSize),
	})
}

// CreateUser creates a user with the given role
//
//	@Summary		Create user
//	@Description	Create a user with one role. Admins may only create role "user"; super admins may create any role.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateUserRequest	true	"User details"
//	@Success		200		{object}	UserDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		403		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	user, err := h.service.Create(c.Request.Context(), middleware.GetUserRolesFromContext(c), req)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(*user)})
}

// UpdateUser updates a user's name and email
//
//	@Summary		Update user
//	@Description	Update name and email of a user the caller is allowed to manage
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"User ID"
//	@Param			request	body		UpdateUserRequest	true	"User details"
//	@Success		200		{object}	UserDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		403		{object}	errs.ErrorResponse
//	@Failure		404		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	user, err := h.service.Update(c.Request.Context(), middleware.GetUserRolesFromContext(c), id, req)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(*user)})
}

// SetPassword sets a new password for a user
//
//	@Summary		Set user password
//	@Description	Set a new password for a user the caller is allowed to manage. Revokes the user's refresh tokens.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"User ID"
//	@Param			request	body		SetPasswordRequest	true	"New password"
//	@Success		200		{object}	internal.MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		403		{object}	errs.ErrorResponse
//	@Failure		404		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/users/{id}/set-password [post]
func (h *Handler) SetPassword(c *gin.Context) {
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.SetPassword(c.Request.Context(), middleware.GetUserRolesFromContext(c), id, req.Password); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Password updated successfully")
}

// DeleteUser deletes a user
//
//	@Summary		Delete user
//	@Description	Delete a user the caller is allowed to manage. Cannot delete yourself.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	internal.MessageResponse
//	@Failure		400	{object}	errs.ErrorResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		403	{object}	errs.ErrorResponse
//	@Failure		404	{object}	errs.ErrorResponse
//	@Failure		500	{object}	errs.ErrorResponse
//	@Router			/api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	id, ok := middleware.PathID(c, "id")
	if !ok {
		return
	}

	callerID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), callerID, middleware.GetUserRolesFromContext(c), id); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "User deleted successfully")
}
