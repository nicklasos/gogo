package unit

import (
	"context"
	"testing"
	"time"

	"app/internal/auth"
	"app/internal/db"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	t.Run("should register new user successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create auth service
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Register user
			req := auth.RegisterRequest{
				Email:    "newuser@example.com",
				Name:     "New User",
				Password: "password123",
			}

			tokenPair, user, err := service.Register(ctx, req)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, tokenPair)
			assert.NotNil(t, user)
			assert.NotEmpty(t, tokenPair.AccessToken)
			assert.NotEmpty(t, tokenPair.RefreshToken)
			assert.Equal(t, req.Email, user.Email)
			assert.Equal(t, req.Name, user.Name)
			assert.True(t, user.ID > 0)
		})
	})

	t.Run("should return error when user already exists", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create existing user
			user := helpers.CreateTestUser(t, ctx, tx)

			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Try to register with same email
			req := auth.RegisterRequest{
				Email:    user.Email,
				Name:     "Another User",
				Password: "password123",
			}

			tokenPair, resultUser, err := service.Register(ctx, req)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrUserAlreadyExists, err)
			assert.Nil(t, tokenPair)
			assert.Nil(t, resultUser)
		})
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Register a user first
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			registerReq := auth.RegisterRequest{
				Email:    "user@example.com",
				Name:     "Test User",
				Password: "password123",
			}

			_, registeredUser, err := service.Register(ctx, registerReq)
			require.NoError(t, err)

			// Test: Login with correct credentials
			loginReq := auth.LoginRequest{
				Email:    "user@example.com",
				Password: "password123",
			}

			tokenPair, user, err := service.Login(ctx, loginReq)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, tokenPair)
			assert.NotNil(t, user)
			assert.NotEmpty(t, tokenPair.AccessToken)
			assert.NotEmpty(t, tokenPair.RefreshToken)
			assert.Equal(t, registeredUser.ID, user.ID)
			assert.Equal(t, registeredUser.Email, user.Email)
		})
	})

	t.Run("should return error with invalid email", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create auth service
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Login with non-existent email
			loginReq := auth.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			}

			tokenPair, user, err := service.Login(ctx, loginReq)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrInvalidCredentials, err)
			assert.Nil(t, tokenPair)
			assert.Nil(t, user)
		})
	})

	t.Run("should return error with invalid password", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Register a user first
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			registerReq := auth.RegisterRequest{
				Email:    "user@example.com",
				Name:     "Test User",
				Password: "password123",
			}

			_, _, err := service.Register(ctx, registerReq)
			require.NoError(t, err)

			// Test: Login with wrong password
			loginReq := auth.LoginRequest{
				Email:    "user@example.com",
				Password: "wrongpassword",
			}

			tokenPair, user, err := service.Login(ctx, loginReq)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrInvalidCredentials, err)
			assert.Nil(t, tokenPair)
			assert.Nil(t, user)
		})
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("should refresh token successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Register a user and get tokens
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			registerReq := auth.RegisterRequest{
				Email:    "user@example.com",
				Name:     "Test User",
				Password: "password123",
			}

			tokenPair, _, err := service.Register(ctx, registerReq)
			require.NoError(t, err)

			// Test: Refresh token
			// Add small delay to ensure different timestamps
			time.Sleep(100 * time.Millisecond)
			newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, newTokenPair)
			assert.NotEmpty(t, newTokenPair.AccessToken)
			assert.NotEmpty(t, newTokenPair.RefreshToken)
			assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken, "New access token should be different from old one")
		})
	})

	t.Run("should return error with invalid refresh token", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create auth service
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Refresh with invalid token
			tokenPair, err := service.RefreshToken(ctx, "invalid-token")

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrInvalidToken, err)
			assert.Nil(t, tokenPair)
		})
	})
}

func TestAuthService_VerifyJWT(t *testing.T) {
	t.Run("should verify valid JWT token", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Register a user and get token
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			registerReq := auth.RegisterRequest{
				Email:    "user@example.com",
				Name:     "Test User",
				Password: "password123",
			}

			tokenPair, _, err := service.Register(ctx, registerReq)
			require.NoError(t, err)

			// Test: Verify token
			token, err := service.VerifyJWT(tokenPair.AccessToken)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, token)
			assert.True(t, token.Valid)
		})
	})

	t.Run("should return error with invalid JWT token", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create auth service
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Verify invalid token
			token, err := service.VerifyJWT("invalid-token")

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrInvalidToken, err)
			assert.Nil(t, token)
		})
	})
}

func TestAuthService_GetUserFromContext(t *testing.T) {
	t.Run("should get user by ID successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)

			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Get user by ID
			resultUser, err := service.GetUserFromContext(ctx, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, resultUser)
			assert.Equal(t, user.ID, resultUser.ID)
			assert.Equal(t, user.Email, resultUser.Email)
		})
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create auth service
			jwtSecret := []byte("test-secret-key")
			testLogger := helpers.GetTestLogger(t)
			service := auth.NewAuthService(queries, jwtSecret, testLogger)

			// Test: Get non-existent user
			user, err := service.GetUserFromContext(ctx, 99999)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, auth.ErrUserNotFound, err)
			assert.Nil(t, user)
		})
	})
}
