package auth

import (
	"app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *AuthService
}

func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func userResponseFromDB(user *db.User) UserResponse {
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
	}
}

// Register creates a new user account
//
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
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {

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
//
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
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, user, err := h.service.Login(c.Request.Context(), req)
	if err != nil {

		errs.RespondWithError(c, err)
		return
	}

	middleware.MarkAuthSuccess(c)

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         userResponseFromDB(user),
	}

	c.JSON(http.StatusOK, LoginDataResponse{Data: response})
}

// RefreshToken refreshes the access token using a refresh token
//
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
		errs.RespondWithValidationError(c, err)
		return
	}

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {

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
//
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
	userIDInt32, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	user, err := h.service.GetUserFromContext(c.Request.Context(), userIDInt32)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(user)})
}

// UpdateMe updates the current authenticated user's name and email
//
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
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req.Email, req.Name)
	if err != nil {
		errs.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserDataResponse{Data: userResponseFromDB(user)})
}

// UpdatePassword updates the current authenticated user's password
//
//	@Summary		Change current user password
//	@Description	Change password for the currently authenticated user. Revokes all refresh tokens.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdatePasswordRequest	true	"Password change"
//	@Success		200		{object}	internal.MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		401		{object}	errs.ErrorResponse
//	@Failure		500		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/me/password [put]
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.UpdatePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Password updated successfully")
}

// ForgotPassword emails a password reset link
//
//	@Summary		Request a password reset
//	@Description	Emails a single-use reset link when the address belongs to an account. Always answers 200, so it does not reveal which emails are registered.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ForgotPasswordRequest	true	"Account email"
//	@Success		200		{object}	internal.MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		429		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	h.service.RequestPasswordReset(c.Request.Context(), req.Email)

	internal.RespondMessage(c, "If the email belongs to an account, a reset link has been sent")
}

// ResetPassword sets a new password from an emailed link
//
//	@Summary		Reset password
//	@Description	Sets a new password with the token from the emailed link. The link works once; all sessions are signed out.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ResetPasswordRequest	true	"Token and new password"
//	@Success		200		{object}	internal.MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		429		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Password updated successfully")
}

// VerifyEmail confirms an email address from an emailed link
//
//	@Summary		Verify email
//	@Description	Confirms an email address with the token from the emailed link
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		VerifyEmailRequest	true	"Token"
//	@Success		200		{object}	internal.MessageResponse
//	@Failure		400		{object}	errs.ErrorResponse
//	@Failure		429		{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Email verified")
}

// ResendVerification sends a new email verification link to the current user
//
//	@Summary		Resend verification email
//	@Description	Sends a new verification link to the current user's email address
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	internal.MessageResponse
//	@Failure		400	{object}	errs.ErrorResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Failure		429	{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/me/verify-email [post]
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	if err := h.service.ResendEmailVerification(c.Request.Context(), userID); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Verification email sent")
}

// RegistrationDisabled answers POST /auth/register when ALLOW_REGISTRATION is off
func RegistrationDisabled(c *gin.Context) {
	errs.RespondWithError(c, errs.NewForbiddenError(errs.ErrKeyAuthRegistrationClosed, "Registration is disabled"))
}

// Logout logs out the current user
//
//	@Summary		Logout user
//	@Description	Logout the currently authenticated user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	internal.MessageResponse
//	@Failure		401	{object}	errs.ErrorResponse
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		return
	}

	if err := h.service.Logout(c.Request.Context(), userID); err != nil {
		errs.RespondWithError(c, err)
		return
	}

	internal.RespondMessage(c, "Logged out successfully")
}
