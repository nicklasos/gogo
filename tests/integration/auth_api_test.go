package integration

import (
	"app/internal/auth"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"
	"app/tests/helpers"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthAPI_Register(t *testing.T) {
	t.Run("should return 200 with tokens when registration is successful", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Register user
			reqBody := `{
				"email": "newuser@example.com",
				"name": "New User",
				"password": "password123"
			}`

			resp := server.POST("/api/v1/auth/register", reqBody)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response auth.RegisterDataResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotEmpty(t, response.Data.AccessToken)
			assert.NotEmpty(t, response.Data.RefreshToken)
			assert.NotNil(t, response.Data.User)
			assert.Equal(t, "newuser@example.com", response.Data.User.Email)
			assert.Equal(t, "New User", response.Data.User.Name)
			assert.Equal(t, []string{middleware.RoleUser}, response.Data.User.Roles)
			assert.True(t, response.Data.User.ID > 0)
		})
	})

	t.Run("should return 400 when email is missing", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Register without email
			reqBody := `{
				"name": "New User",
				"password": "password123"
			}`

			resp := server.POST("/api/v1/auth/register", reqBody)

			// Assert: Check response status
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	})

	t.Run("should return 400 when user already exists", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Register first user
			reqBody := `{
				"email": "existing@example.com",
				"name": "Existing User",
				"password": "password123"
			}`

			resp := server.POST("/api/v1/auth/register", reqBody)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Test: Try to register with same email
			resp2 := server.POST("/api/v1/auth/register", reqBody)

			// Assert: Check response status
			assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

			// Assert: Parse and validate error response
			var errorResponse map[string]interface{}
			err := resp2.JSON(&errorResponse)
			require.NoError(t, err)

			assert.Equal(t, "auth.user_exists", errorResponse["error_key"])
		})
	})
}

func TestAuthAPI_Login(t *testing.T) {
	t.Run("should return 200 with tokens when login is successful", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Register user
			registerReq := `{
				"email": "user@example.com",
				"name": "Test User",
				"password": "password123"
			}`

			regResp := server.POST("/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusOK, regResp.StatusCode)

			// Test: Login
			loginReq := `{
				"email": "user@example.com",
				"password": "password123"
			}`

			resp := server.POST("/api/v1/auth/login", loginReq)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response auth.LoginDataResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotEmpty(t, response.Data.AccessToken)
			assert.NotEmpty(t, response.Data.RefreshToken)
			assert.NotNil(t, response.Data.User)
			assert.Equal(t, "user@example.com", response.Data.User.Email)
			assert.True(t, response.Data.User.ID > 0)
		})
	})

	t.Run("should return 401 when password is wrong", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Register user
			registerReq := `{
				"email": "user@example.com",
				"name": "Test User",
				"password": "password123"
			}`

			regResp := server.POST("/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusOK, regResp.StatusCode)

			// Test: Login with wrong password
			loginReq := `{
				"email": "user@example.com",
				"password": "wrongpassword"
			}`

			resp := server.POST("/api/v1/auth/login", loginReq)

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			// Assert: Parse and validate error response
			var errorResponse map[string]interface{}
			err := resp.JSON(&errorResponse)
			require.NoError(t, err)

			assert.Equal(t, "auth.invalid_credentials", errorResponse["error_key"])
		})
	})

	t.Run("should return 401 when user does not exist", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Login with non-existent user
			loginReq := `{
				"email": "nonexistent@example.com",
				"password": "password123"
			}`

			resp := server.POST("/api/v1/auth/login", loginReq)

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			// Assert: Parse and validate error response
			var errorResponse map[string]interface{}
			err := resp.JSON(&errorResponse)
			require.NoError(t, err)

			assert.Equal(t, "auth.invalid_credentials", errorResponse["error_key"])
		})
	})
}

func TestAuthAPI_RefreshToken(t *testing.T) {
	t.Run("should return 200 with new tokens when refresh is successful", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Register user and get tokens
			registerReq := `{
				"email": "user@example.com",
				"name": "Test User",
				"password": "password123"
			}`

			regResp := server.POST("/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusOK, regResp.StatusCode)

			var registerResponse auth.RegisterDataResponse
			err := regResp.JSON(&registerResponse)
			require.NoError(t, err)

			// Test: Refresh token
			// Add small delay to ensure different timestamps
			time.Sleep(100 * time.Millisecond)
			refreshReq := `{
				"refresh_token": "` + registerResponse.Data.RefreshToken + `"
			}`

			resp := server.POST("/api/v1/auth/refresh", refreshReq)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response auth.RefreshTokenDataResponse
			err = resp.JSON(&response)
			require.NoError(t, err)

			assert.NotEmpty(t, response.Data.AccessToken)
			assert.NotEmpty(t, response.Data.RefreshToken)
			assert.NotEqual(t, registerResponse.Data.AccessToken, response.Data.AccessToken, "New access token should be different from old one")
		})
	})

	t.Run("should return 401 when refresh token is invalid", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Refresh with invalid token
			refreshReq := `{
				"refresh_token": "invalid-token"
			}`

			resp := server.POST("/api/v1/auth/refresh", refreshReq)

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			// Assert: Parse and validate error response
			var errorResponse map[string]interface{}
			err := resp.JSON(&errorResponse)
			require.NoError(t, err)

			assert.Equal(t, "auth.invalid_token", errorResponse["error_key"])
		})
	})
}

func TestAuthAPI_GetMe(t *testing.T) {
	t.Run("should return 200 with user info when authenticated", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Setup: Register user and get token
			registerReq := `{
				"email": "user@example.com",
				"name": "Test User",
				"password": "password123"
			}`

			regResp := server.POST("/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusOK, regResp.StatusCode)

			var registerResponse auth.RegisterDataResponse
			err := regResp.JSON(&registerResponse)
			require.NoError(t, err)

			// Test: Get current user
			req := server.NewRequest("GET", "/api/v1/auth/me", nil)
			req.Header.Set("Authorization", "Bearer "+registerResponse.Data.AccessToken)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response auth.UserDataResponse
			err = resp.JSON(&response)
			require.NoError(t, err)

			assert.Equal(t, "user@example.com", response.Data.Email)
			assert.Equal(t, "Test User", response.Data.Name)
			assert.True(t, response.Data.ID > 0)
		})
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Get current user without token
			resp := server.GET("/api/v1/auth/me")

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	})
}

func TestAuthAPI_GetMe_Roles(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := helpers.CreateTestUserWithRoles(t, ctx, tx, middleware.RoleSuperAdmin)
		token := helpers.GenerateTestJWT(user.ID, user.Email)

		resp := server.GETAuth("/api/v1/auth/me", token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response auth.UserDataResponse
		require.NoError(t, resp.JSON(&response))
		assert.Equal(t, []string{middleware.RoleSuperAdmin}, response.Data.Roles)
	})
}

func TestAuthAPI_UpdateMe(t *testing.T) {
	t.Run("should update name and email", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := helpers.CreateTestUser(t, ctx, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			resp := server.PUTAuth("/api/v1/auth/me", `{"email": "updated@example.com", "name": "Updated"}`, token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response auth.UserDataResponse
			require.NoError(t, resp.JSON(&response))
			assert.Equal(t, "updated@example.com", response.Data.Email)
			assert.Equal(t, "Updated", response.Data.Name)
		})
	})

	t.Run("should return 400 when email is taken", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			other := helpers.CreateTestUserWithEmail(t, ctx, tx, "taken@example.com")
			user := helpers.CreateTestUser(t, ctx, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			resp := server.PUTAuth("/api/v1/auth/me", fmt.Sprintf(`{"email": %q, "name": "Updated"}`, other.Email), token)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var body errs.ErrorResponse
			require.NoError(t, resp.JSON(&body))
			assert.Equal(t, errs.ErrKeyAuthUserExists, body.ErrorKey)
		})
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			resp := server.PUT("/api/v1/auth/me", `{"email": "x@example.com", "name": "X"}`)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	})
}

func TestAuthAPI_UpdatePassword(t *testing.T) {
	t.Run("should change the password and revoke refresh tokens", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := helpers.CreateTestUser(t, ctx, tx)
			login := func(password string) *helpers.TestResponse {
				return server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": %q}`, user.Email, password))
			}

			var session auth.LoginDataResponse
			require.NoError(t, login("password123").JSON(&session))

			resp := server.PUTAuth("/api/v1/auth/me/password", `{"current_password": "password123", "new_password": "new-password-1"}`, session.Data.AccessToken)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, http.StatusUnauthorized, login("password123").StatusCode)
			assert.Equal(t, http.StatusOK, login("new-password-1").StatusCode)

			refresh := server.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken))
			assert.Equal(t, http.StatusUnauthorized, refresh.StatusCode)
		})
	})

	t.Run("should return 400, not 401, when the current password is wrong", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := helpers.CreateTestUser(t, ctx, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			resp := server.PUTAuth("/api/v1/auth/me/password", `{"current_password": "wrong", "new_password": "new-password-1"}`, token)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var body errs.ErrorResponse
			require.NoError(t, resp.JSON(&body))
			assert.Equal(t, errs.ErrKeyAuthInvalidCurrentPass, body.ErrorKey)
		})
	})
}

func TestAuthAPI_RefreshTokenIsSingleUse(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := helpers.CreateTestUser(t, ctx, tx)

		var session auth.LoginDataResponse
		resp := server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
		require.NoError(t, resp.JSON(&session))

		body := fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken)
		assert.Equal(t, http.StatusOK, server.POST("/api/v1/auth/refresh", body).StatusCode)
		assert.Equal(t, http.StatusUnauthorized, server.POST("/api/v1/auth/refresh", body).StatusCode)
	})
}

func TestRequestID(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		assert.NotEmpty(t, server.GET("/health").Header.Get("X-Request-ID"))

		req := server.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Request-ID", "abc-123_ok.1")
		assert.Equal(t, "abc-123_ok.1", server.Do(req).Header.Get("X-Request-ID"))

		req = server.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Request-ID", "bad value\twith spaces")
		got := server.Do(req).Header.Get("X-Request-ID")
		assert.NotEmpty(t, got)
		assert.NotEqual(t, "bad value\twith spaces", got)
	})
}
