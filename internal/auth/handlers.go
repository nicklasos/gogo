package auth

import (
	"app/internal/db"
	"app/internal/errs"
	"app/internal/logger"
	"app/internal/middleware"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *AuthService
	logger  *logger.Logger
}

func NewAuthHandler(service *AuthService, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

func userResponseFromDB(user *db.User) UserResponse {
	roles := user.Roles
	if roles == nil {
		roles = []string{}
	}
	return UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Roles: roles,
	}
}

// Register creates a new user account
//	@Summary		Register new user
//	@Description	Create a new user account with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration request"
//	@Success		200		{object}	RegisterDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Invalid request body", "error", err)
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to register user", "error", err, "email", req.Email)

		errs.RespondWithError(c, err)
		return
	}

	response := RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         userResponseFromDB(user),
	}

	c.JSON(http.StatusOK, RegisterDataResponse{Data: response})
}

// Login authenticates a user
//	@Summary		Login user
//	@Description	Authenticate user with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login request"
//	@Success		200		{object}	LoginDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Invalid request body", "error", err)
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, user, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to login", "error", err, "email", req.Email)

		errs.RespondWithError(c, err)
		return
	}

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         userResponseFromDB(user),
	}

	c.JSON(http.StatusOK, LoginDataResponse{Data: response})
}

// RefreshToken refreshes the access token using a refresh token
//	@Summary		Refresh access token
//	@Description	Refresh the access token using a valid refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RefreshTokenRequest	true	"Refresh token request"
//	@Success		200		{object}	RefreshTokenDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Invalid request body", "error", err)
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to refresh token", "error", err)

		errs.RespondWithError(c, err)
		return
	}

	response := RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}

	c.JSON(http.StatusOK, RefreshTokenDataResponse{Data: response})
}

// GetMe returns the current authenticated user's information
//	@Summary		Get current user info
//	@Description	Get information about the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserDataResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		500	{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userIDInt32, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		if errors.Is(err, middleware.ErrUserNotAuthenticated) {
			errs.RespondWithUnauthorized(c, "Unauthorized")
		} else {
			errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid user ID format")
		}
		return
	}

	user, err := h.service.GetUserFromContext(c.Request.Context(), userIDInt32)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to get user", "error", err, "user_id", userIDInt32)
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(user)})
}

// UpdateMe updates the current authenticated user's name and email
//	@Summary		Update current user
//	@Description	Update name and email for the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateProfileRequest	true	"Profile update"
//	@Success		200		{object}	UserDataResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/me [put]
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req.Email, req.Name)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update profile", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(user)})
}

// UpdatePassword updates the current authenticated user's password
//	@Summary		Change current user password
//	@Description	Change password for the currently authenticated user. Revokes all refresh tokens.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdatePasswordRequest	true	"Password change"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/me/password [put]
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.UpdatePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to update password", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	var response MessageResponse
	response.Data.Message = "Password updated successfully"
	c.JSON(http.StatusOK, response)
}

// Logout logs out the current user
//	@Summary		Logout user
//	@Description	Logout the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	MessageResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithUnauthorized(c, "Unauthorized")
		return
	}

	if err := h.service.Logout(c.Request.Context(), userID); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "Failed to logout", "error", err, "user_id", userID)
		errs.RespondWithError(c, err)
		return
	}

	var response MessageResponse
	response.Data.Message = "Logged out successfully"
	c.JSON(http.StatusOK, response)
}
