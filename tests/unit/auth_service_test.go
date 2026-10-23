package unit

import (
	"testing"
	"time"

	"app/internal/auth"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthService(app *testkit.App) *auth.AuthService {
	return auth.NewAuthService(app.Queries, app.TxRunner, app.Mail, app.Logger, auth.Options{JWTSecret: []byte(app.Config.JWTSecret)})
}

func TestAuthService_Register(t *testing.T) {
	t.Run("should register new user successfully", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		req := auth.RegisterRequest{
			Email:    "newuser@example.com",
			Name:     "New User",
			Password: "password123",
		}

		tokenPair, user, err := service.Register(app.Ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.NotNil(t, user)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Name, user.Name)
		assert.True(t, user.ID > 0)
	})

	t.Run("should return error when user already exists", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		service := newAuthService(app)

		req := auth.RegisterRequest{
			Email:    user.Email,
			Name:     "Another User",
			Password: "password123",
		}

		tokenPair, resultUser, err := service.Register(app.Ctx, req)

		assert.Error(t, err)
		assert.Equal(t, auth.ErrUserAlreadyExists, err)
		assert.Nil(t, tokenPair)
		assert.Nil(t, resultUser)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		registerReq := auth.RegisterRequest{
			Email:    "user@example.com",
			Name:     "Test User",
			Password: "password123",
		}

		_, registeredUser, err := service.Register(app.Ctx, registerReq)
		require.NoError(t, err)

		loginReq := auth.LoginRequest{
			Email:    "user@example.com",
			Password: "password123",
		}

		tokenPair, user, err := service.Login(app.Ctx, loginReq)

		require.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.NotNil(t, user)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, registeredUser.ID, user.ID)
		assert.Equal(t, registeredUser.Email, user.Email)
	})

	t.Run("should return error with invalid email", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		loginReq := auth.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		tokenPair, user, err := service.Login(app.Ctx, loginReq)

		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidCredentials, err)
		assert.Nil(t, tokenPair)
		assert.Nil(t, user)
	})

	t.Run("should return error with invalid password", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		registerReq := auth.RegisterRequest{
			Email:    "user@example.com",
			Name:     "Test User",
			Password: "password123",
		}

		_, _, err := service.Register(app.Ctx, registerReq)
		require.NoError(t, err)

		loginReq := auth.LoginRequest{
			Email:    "user@example.com",
			Password: "wrongpassword",
		}

		tokenPair, user, err := service.Login(app.Ctx, loginReq)

		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidCredentials, err)
		assert.Nil(t, tokenPair)
		assert.Nil(t, user)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("should refresh token successfully", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		registerReq := auth.RegisterRequest{
			Email:    "user@example.com",
			Name:     "Test User",
			Password: "password123",
		}

		tokenPair, _, err := service.Register(app.Ctx, registerReq)
		require.NoError(t, err)

		// Add small delay to ensure different timestamps
		time.Sleep(100 * time.Millisecond)
		newTokenPair, err := service.RefreshToken(app.Ctx, tokenPair.RefreshToken)

		require.NoError(t, err)
		assert.NotNil(t, newTokenPair)
		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken, "New access token should be different from old one")
	})

	t.Run("should return error with invalid refresh token", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		tokenPair, err := service.RefreshToken(app.Ctx, "invalid-token")

		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidToken, err)
		assert.Nil(t, tokenPair)
	})
}

func TestAuthService_VerifyJWT(t *testing.T) {
	t.Run("should verify valid JWT token", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		registerReq := auth.RegisterRequest{
			Email:    "user@example.com",
			Name:     "Test User",
			Password: "password123",
		}

		tokenPair, _, err := service.Register(app.Ctx, registerReq)
		require.NoError(t, err)

		token, err := service.VerifyJWT(tokenPair.AccessToken)

		require.NoError(t, err)
		assert.NotNil(t, token)
		assert.True(t, token.Valid)
	})

	t.Run("should return error with invalid JWT token", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		token, err := service.VerifyJWT("invalid-token")

		assert.Error(t, err)
		assert.Equal(t, auth.ErrInvalidToken, err)
		assert.Nil(t, token)
	})
}

func TestAuthService_GetUserFromContext(t *testing.T) {
	t.Run("should get user by ID successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		service := newAuthService(app)

		resultUser, err := service.GetUserFromContext(app.Ctx, user.ID)

		require.NoError(t, err)
		assert.NotNil(t, resultUser)
		assert.Equal(t, user.ID, resultUser.ID)
		assert.Equal(t, user.Email, resultUser.Email)
	})

	t.Run("should return error when user not found", func(t *testing.T) {
		app := testkit.New(t)
		service := newAuthService(app)

		user, err := service.GetUserFromContext(app.Ctx, 99999)

		assert.Error(t, err)
		assert.Equal(t, auth.ErrUserNotFound, err)
		assert.Nil(t, user)
	})
}
