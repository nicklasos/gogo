package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"app/internal/auth"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/middleware"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
)

func TestAuthAPI_Register(t *testing.T) {
	t.Run("should return 200 with tokens when registration is successful", func(t *testing.T) {
		app := testkit.New(t)
		reqBody := `{
			"email": "newuser@example.com",
			"name": "New User",
			"password": "password123"
		}`

		resp := app.POST("/api/v1/auth/register", reqBody)

		resp.AssertOK()

		var response auth.RegisterDataResponse
		resp.JSON(&response)

		assert.NotEmpty(t, response.Data.AccessToken)
		assert.NotEmpty(t, response.Data.RefreshToken)
		assert.NotNil(t, response.Data.User)
		assert.Equal(t, "newuser@example.com", response.Data.User.Email)
		assert.Equal(t, "New User", response.Data.User.Name)
		assert.Equal(t, []string{middleware.RoleUser}, response.Data.User.Roles)
		assert.True(t, response.Data.User.ID > 0)
	})

	t.Run("should return 400 when email is missing", func(t *testing.T) {
		app := testkit.New(t)
		reqBody := `{
			"name": "New User",
			"password": "password123"
		}`

		resp := app.POST("/api/v1/auth/register", reqBody)

		resp.AssertBadRequest()
	})

	t.Run("should return 400 when user already exists", func(t *testing.T) {
		app := testkit.New(t)
		reqBody := `{
			"email": "existing@example.com",
			"name": "Existing User",
			"password": "password123"
		}`

		app.POST("/api/v1/auth/register", reqBody).AssertOK()

		resp2 := app.POST("/api/v1/auth/register", reqBody)

		resp2.AssertBadRequest()

		var errorResponse map[string]interface{}
		resp2.JSON(&errorResponse)

		assert.Equal(t, "auth.user_exists", errorResponse["error_key"])
	})
}

func TestAuthAPI_Login(t *testing.T) {
	t.Run("should return 200 with tokens when login is successful", func(t *testing.T) {
		app := testkit.New(t)
		registerReq := `{
			"email": "user@example.com",
			"name": "Test User",
			"password": "password123"
		}`

		regResp := app.POST("/api/v1/auth/register", registerReq)
		regResp.AssertOK()

		loginReq := `{
			"email": "user@example.com",
			"password": "password123"
		}`

		resp := app.POST("/api/v1/auth/login", loginReq)

		resp.AssertOK()

		var response auth.LoginDataResponse
		resp.JSON(&response)

		assert.NotEmpty(t, response.Data.AccessToken)
		assert.NotEmpty(t, response.Data.RefreshToken)
		assert.NotNil(t, response.Data.User)
		assert.Equal(t, "user@example.com", response.Data.User.Email)
		assert.True(t, response.Data.User.ID > 0)
	})

	t.Run("should return 401 when password is wrong", func(t *testing.T) {
		app := testkit.New(t)
		registerReq := `{
			"email": "user@example.com",
			"name": "Test User",
			"password": "password123"
		}`

		regResp := app.POST("/api/v1/auth/register", registerReq)
		regResp.AssertOK()

		loginReq := `{
			"email": "user@example.com",
			"password": "wrongpassword"
		}`

		resp := app.POST("/api/v1/auth/login", loginReq)

		resp.AssertUnauthorized()

		var errorResponse map[string]interface{}
		resp.JSON(&errorResponse)

		assert.Equal(t, "auth.invalid_credentials", errorResponse["error_key"])
	})

	t.Run("should return 401 when user does not exist", func(t *testing.T) {
		app := testkit.New(t)
		loginReq := `{
			"email": "nonexistent@example.com",
			"password": "password123"
		}`

		resp := app.POST("/api/v1/auth/login", loginReq)

		resp.AssertUnauthorized()

		var errorResponse map[string]interface{}
		resp.JSON(&errorResponse)

		assert.Equal(t, "auth.invalid_credentials", errorResponse["error_key"])
	})
}

func TestAuthAPI_RefreshToken(t *testing.T) {
	t.Run("should return 200 with new tokens when refresh is successful", func(t *testing.T) {
		app := testkit.New(t)
		registerReq := `{
			"email": "user@example.com",
			"name": "Test User",
			"password": "password123"
		}`

		regResp := app.POST("/api/v1/auth/register", registerReq)
		regResp.AssertOK()

		var registerResponse auth.RegisterDataResponse
		regResp.JSON(&registerResponse)

		// Add small delay to ensure different timestamps
		time.Sleep(100 * time.Millisecond)
		refreshReq := `{
			"refresh_token": "` + registerResponse.Data.RefreshToken + `"
		}`

		resp := app.POST("/api/v1/auth/refresh", refreshReq)

		resp.AssertOK()

		var response auth.RefreshTokenDataResponse
		resp.JSON(&response)

		assert.NotEmpty(t, response.Data.AccessToken)
		assert.NotEmpty(t, response.Data.RefreshToken)
		assert.NotEqual(t, registerResponse.Data.AccessToken, response.Data.AccessToken, "New access token should be different from old one")
	})

	t.Run("should return 401 when refresh token is invalid", func(t *testing.T) {
		app := testkit.New(t)
		refreshReq := `{
			"refresh_token": "invalid-token"
		}`

		resp := app.POST("/api/v1/auth/refresh", refreshReq)

		resp.AssertUnauthorized()

		var errorResponse map[string]interface{}
		resp.JSON(&errorResponse)

		assert.Equal(t, "auth.invalid_token", errorResponse["error_key"])
	})
}

func TestAuthAPI_GetMe(t *testing.T) {
	t.Run("should return 200 with user info when authenticated", func(t *testing.T) {
		app := testkit.New(t)
		registerReq := `{
			"email": "user@example.com",
			"name": "Test User",
			"password": "password123"
		}`

		regResp := app.POST("/api/v1/auth/register", registerReq)
		regResp.AssertOK()

		var registerResponse auth.RegisterDataResponse
		regResp.JSON(&registerResponse)

		app.WithToken(registerResponse.Data.AccessToken).GET("/api/v1/auth/me").
			AssertOK().
			AssertJSONPath("data.email", "user@example.com").
			AssertJSONPath("data.name", "Test User")
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		app := testkit.New(t)
		resp := app.GET("/api/v1/auth/me")

		resp.AssertUnauthorized()
	})
}

func TestAuthAPI_GetMe_Roles(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User(factory.WithRoles(middleware.RoleSuperAdmin))
	token := app.Token(user)

	resp := app.WithToken(token).GET("/api/v1/auth/me").AssertOK()

	var response auth.UserDataResponse
	resp.JSON(&response)
	assert.Equal(t, []string{middleware.RoleSuperAdmin}, response.Data.Roles)
}

func TestAuthAPI_UpdateMe(t *testing.T) {
	t.Run("should update name and email", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		token := app.Token(user)

		resp := app.WithToken(token).PUT("/api/v1/auth/me", `{"email": "updated@example.com", "name": "Updated"}`).AssertOK()

		var response auth.UserDataResponse
		resp.JSON(&response)
		assert.Equal(t, "updated@example.com", response.Data.Email)
		assert.Equal(t, "Updated", response.Data.Name)
	})

	t.Run("should return 400 when email is taken", func(t *testing.T) {
		app := testkit.New(t)
		other := app.Create.User(factory.WithEmail("taken@example.com"))
		user := app.Create.User()
		token := app.Token(user)

		resp := app.WithToken(token).PUT("/api/v1/auth/me", fmt.Sprintf(`{"email": %q, "name": "Updated"}`, other.Email)).AssertBadRequest()

		var body errs.ErrorResponse
		resp.JSON(&body)
		assert.Equal(t, errs.ErrKeyAuthUserExists, body.ErrorKey)
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		app := testkit.New(t)
		app.PUT("/api/v1/auth/me", `{"email": "x@example.com", "name": "X"}`).AssertUnauthorized()
	})
}

func TestAuthAPI_UpdatePassword(t *testing.T) {
	t.Run("should change the password and revoke refresh tokens", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		var session auth.LoginDataResponse
		login(app, user.Email, "password123").JSON(&session)

		app.WithToken(session.Data.AccessToken).PUT("/api/v1/auth/me/password", `{"current_password": "password123", "new_password": "new-password-1"}`).AssertOK()

		login(app, user.Email, "password123").AssertUnauthorized()
		login(app, user.Email, "new-password-1").AssertOK()

		app.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken)).AssertUnauthorized()
	})

	t.Run("should return 400, not 401, when the current password is wrong", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).PUT("/api/v1/auth/me/password", `{"current_password": "wrong", "new_password": "new-password-1"}`).
			AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidCurrentPass)
	})
}

func TestAuthAPI_RefreshTokenIsSingleUse(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()

	var session auth.LoginDataResponse
	resp := app.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
	resp.JSON(&session)

	body := fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken)
	app.POST("/api/v1/auth/refresh", body).AssertOK()
	app.POST("/api/v1/auth/refresh", body).AssertUnauthorized()
}

func TestRequestID(t *testing.T) {
	app := testkit.New(t)
	assert.NotEmpty(t, app.GET("/health").Header.Get("X-Request-ID"))

	kept := app.WithHeader("X-Request-ID", "abc-123_ok.1").GET("/health")
	assert.Equal(t, "abc-123_ok.1", kept.Header.Get("X-Request-ID"))

	got := app.WithHeader("X-Request-ID", "bad value\twith spaces").GET("/health").Header.Get("X-Request-ID")
	assert.NotEmpty(t, got)
	assert.NotEqual(t, "bad value\twith spaces", got)
}
