package auth

import (
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

// Register creates a new user account
//	@Summary		Register new user
//	@Description	Create a new user account with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration request"
//	@Success		200		{object}	RegisterDataResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
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

		switch err {
		case ErrUserAlreadyExists:
			errs.RespondWithError(c, err)
		default:
			errs.RespondWithError(c, err)
		}
		return
	}

	response := RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
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
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
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

		switch err {
		case ErrInvalidCredentials:
			errs.RespondWithError(c, err)
		default:
			errs.RespondWithError(c, err)
		}
		return
	}

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
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
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
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

		switch err {
		case ErrInvalidToken:
			errs.RespondWithError(c, err)
		case ErrUserNotFound:
			errs.RespondWithError(c, err)
		default:
			errs.RespondWithError(c, err)
		}
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
//	@Security		Bearer
//	@Success		200	{object}	UserDataResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
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

	response := UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: response})
}

// Logout logs out the current user
//	@Summary		Logout user
//	@Description	Logout the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	MessageResponse
//	@Failure		401	{object}	ErrorResponse
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var response MessageResponse
	response.Data.Message = "Logged out successfully"
	c.JSON(http.StatusOK, response)
}
