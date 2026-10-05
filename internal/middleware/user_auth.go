package middleware

import (
	"app/internal/errs"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrUserNotAuthenticated = errs.NewUnauthorizedError(errs.ErrKeyAuthTokenRequired, "User not authenticated")
	ErrInvalidUserIDFormat  = errs.NewBadRequestError(errs.ErrKeyBadRequest, "Invalid user ID format")
)

// Claims represents JWT claims for user authentication
type Claims struct {
	UserID int32  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// UserJWTVerifier interface for JWT verification
type UserJWTVerifier interface {
	VerifyJWT(tokenString string) (*jwt.Token, error)
}

// ExtractBearerToken extracts the Bearer token from the Authorization header
// Returns empty string if no Authorization header is provided
// Returns the token string if found, or empty string if format is invalid
func ExtractBearerToken(authHeader string) (string, bool) {
	if authHeader == "" {
		return "", false
	}

	// Handle both "Bearer <token>" and raw token formats (for Swagger UI compatibility)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return authHeader[7:], true
	}

	// Raw token format (for Swagger UI compatibility)
	return authHeader, true
}

// ExtractUserIDFromJWT extracts the user ID from the JWT in the Authorization header.
// The token is not read from the URL: query strings end up in access logs and browser history.
// Returns (nil, nil) if no token is provided
// Returns (nil, error) if token is invalid or verification fails
// Returns (userID, nil) if token is valid
func ExtractUserIDFromJWT(c *gin.Context, verifier UserJWTVerifier) (*int32, error) {
	tokenString, _ := ExtractBearerToken(c.GetHeader("Authorization"))

	if tokenString == "" {
		return nil, nil
	}

	token, err := verifier.VerifyJWT(tokenString)
	if err != nil {
		return nil, errs.WrapDomainError(errs.ErrKeyAuthInvalidToken, "Invalid or expired token", 401, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidToken, "Invalid token claims")
	}

	return &claims.UserID, nil
}

// UserAuthMiddleware validates the JWT, then loads current roles from the database
// onto the context. RequireRole reads those roles; do not trust JWT claims for them.
func UserAuthMiddleware(auth UserAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := ExtractUserIDFromJWT(c, auth)
		if err != nil {
			errs.RespondWithError(c, err)
			c.Abort()
			return
		}
		if userID == nil {
			errs.RespondWithError(c, ErrUserNotAuthenticated)
			c.Abort()
			return
		}

		roles, err := auth.GetUserRoles(c.Request.Context(), *userID)
		if err != nil {
			errs.RespondWithError(c, err)
			c.Abort()
			return
		}
		if roles == nil {
			roles = []string{}
		}

		c.Set("user_id", *userID)
		c.Set("user_roles", roles)

		c.Next()
	}
}

// OptionalUserAuthMiddleware extracts JWT token and sets user context if present
// Unlike UserAuthMiddleware, this does NOT abort if no token is provided
// This allows routes to be accessible to both authenticated and unauthenticated users
func OptionalUserAuthMiddleware(verifier UserJWTVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := ExtractUserIDFromJWT(c, verifier)

		// If token is present and valid, set user context
		if err == nil && userID != nil {
			c.Set("user_id", *userID)
		}

		// Continue regardless of authentication status
		c.Next()
	}
}

// GetUserIDFromContext retrieves the user ID from the Gin context
// This should be used instead of duplicated user ID extraction code
// Returns a structured error if user is not authenticated or ID format is invalid
func GetUserIDFromContext(c *gin.Context) (int32, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, ErrUserNotAuthenticated
	}

	userIDInt32, ok := userID.(int32)
	if !ok {
		return 0, ErrInvalidUserIDFormat
	}

	return userIDInt32, nil
}

// GetUserRolesFromContext returns roles loaded by UserAuthMiddleware.
func GetUserRolesFromContext(c *gin.Context) []string {
	roles, exists := c.Get("user_roles")
	if !exists {
		return nil
	}
	if list, ok := roles.([]string); ok {
		return list
	}
	return nil
}
