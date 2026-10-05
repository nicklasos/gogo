package auth

// RegisterRequest represents the request structure for user registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents the request structure for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse represents user information
type UserResponse struct {
	ID            int32    `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Roles         []string `json:"roles"`
	EmailVerified bool     `json:"email_verified"`
}

// ForgotPasswordRequest asks for a password reset link
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest sets a new password with the token from the emailed link
type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// VerifyEmailRequest confirms an email address with the token from the emailed link
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// UpdateProfileRequest represents the request to update the current user
type UpdateProfileRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

// UpdatePasswordRequest represents the request to change the current user's password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// RegisterResponse represents the response structure for register endpoint
type RegisterResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// LoginResponse represents the response structure for login endpoint
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents the response structure for refresh token endpoint
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RegisterDataResponse wraps register response in data field
type RegisterDataResponse struct {
	Data RegisterResponse `json:"data"`
}

// LoginDataResponse wraps login response in data field
type LoginDataResponse struct {
	Data LoginResponse `json:"data"`
}

// RefreshTokenDataResponse wraps refresh token response in data field
type RefreshTokenDataResponse struct {
	Data RefreshTokenResponse `json:"data"`
}

// UserDataResponse wraps user data in response
type UserDataResponse struct {
	Data UserResponse `json:"data"`
}
