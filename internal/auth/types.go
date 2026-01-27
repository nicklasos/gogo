package auth

// UserResponse represents user information
type UserResponse struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
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

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error string `json:"error"`
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

// MessageResponse wraps a simple message in response
type MessageResponse struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// UserDataResponse wraps user data in response
type UserDataResponse struct {
	Data UserResponse `json:"data"`
}
