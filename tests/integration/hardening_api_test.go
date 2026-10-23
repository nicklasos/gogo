package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal"
	"app/internal/auth"
	"app/internal/cache"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/health"
	"app/tests/testkit"

	"app/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "user@example.com", internal.NormalizeEmail("  User@Example.COM "))
	assert.Equal(t, "", internal.NormalizeEmail("   "))
}

func TestAuthAPI_EmailsAreCaseInsensitive(t *testing.T) {
	t.Run("login, reset and duplicates ignore case", func(t *testing.T) {
		app := testkit.New(t)
		var registered auth.RegisterDataResponse
		resp := app.POST("/api/v1/auth/register", `{"email": "Mixed.Case@Example.com", "name": "Mixed", "password": "password123"}`).AssertOK()
		resp.JSON(&registered)
		assert.Equal(t, "mixed.case@example.com", registered.Data.User.Email, "stored lower-case")

		for _, email := range []string{"mixed.case@example.com", "MIXED.CASE@EXAMPLE.COM", "Mixed.Case@Example.com"} {
			login(app, email, "password123").AssertOK()
		}

		sentBefore := len(app.Mail.Sent())
		app.POST("/api/v1/auth/forgot-password", `{"email": "MIXED.CASE@example.com"}`)
		require.Len(t, app.Mail.Sent(), sentBefore+1)
		assert.Equal(t, []string{"mixed.case@example.com"}, app.Mail.Sent()[sentBefore].To)

		resp = app.POST("/api/v1/auth/register", `{"email": "MIXED.case@example.com", "name": "Again", "password": "password123"}`).AssertError(http.StatusBadRequest, errs.ErrKeyAuthUserExists)
	})

	t.Run("an account stored with capitals before the change can still sign in", func(t *testing.T) {
		app := testkit.New(t)
		app.Create.User(factory.WithEmail("Legacy.User@example.com"))

		login(app, "legacy.user@example.com", "password123").AssertOK()
		login(app, "Legacy.User@example.com", "password123").AssertOK()
	})

	t.Run("the database refuses two accounts that differ only by case", func(t *testing.T) {
		app := testkit.New(t)
		app.Create.User(factory.WithEmail("twin@example.com"))

		_, err := app.Tx.Exec(app.Ctx, "INSERT INTO users (email, name, password) VALUES ('TWIN@example.com', 'Twin', 'x')")
		require.Error(t, err)
		assert.NotNil(t, errs.DomainErrorFromPostgresUniqueViolation(err), "mapped to the user-exists error")
	})

	t.Run("admins create and edit users with normalised emails", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.SuperAdmin())
		token := app.Token(admin)

		var created struct {
			Data struct {
				ID    int32  `json:"id"`
				Email string `json:"email"`
			} `json:"data"`
		}
		resp := app.WithToken(token).POST("/api/v1/users", `{"email": "New.Person@Example.com", "name": "New", "password": "password123", "role": "user"}`)
		resp.JSON(&created)
		assert.Equal(t, "new.person@example.com", created.Data.Email)

		resp = app.WithToken(token).PUT("/api/v1/users/%d", `{"email": "Renamed.Person@Example.com", "name": "New"}`, created.Data.ID)
		resp.JSON(&created)
		assert.Equal(t, "renamed.person@example.com", created.Data.Email)
	})
}

func TestAuthAPI_RefreshTokensAreStoredHashed(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()

	var session auth.LoginDataResponse
	resp := app.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
	resp.JSON(&session)
	raw := session.Data.RefreshToken
	require.NotEmpty(t, raw)

	var stored string
	require.NoError(t, app.Tx.QueryRow(app.Ctx, "SELECT token FROM refresh_tokens WHERE user_id = $1", user.ID).Scan(&stored))
	sum := sha256.Sum256([]byte(raw))
	assert.Equal(t, hex.EncodeToString(sum[:]), stored)
	assert.NotEqual(t, raw, stored)

	refresh := func(token string) int {
		return app.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, token)).StatusCode
	}
	assert.Equal(t, http.StatusUnauthorized, refresh(stored), "the stored value is not a usable token")
	assert.Equal(t, http.StatusOK, refresh(raw))
	assert.Equal(t, http.StatusUnauthorized, refresh(raw), "still single-use")
}

func TestAuth_TokenInQueryStringIsIgnored(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()

	app.GET("/api/v1/auth/me?token=" + app.Token(user)).AssertUnauthorized()
	app.As(user).GET("/api/v1/auth/me").AssertOK()
}

type brokenCache struct {
	*cache.MemoryCache
}

func (brokenCache) Ping(context.Context) error { return errors.New("connection refused") }

type healthBody struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func TestHealth(t *testing.T) {
	t.Run("reports the database, and the cache only when it has a server behind it", func(t *testing.T) {
		app := testkit.New(t)
		for _, path := range []string{"/health", "/api/v1/health"} {
			var body healthBody
			app.GET(path).AssertOK().JSON(&body)
			assert.Equal(t, "healthy", body.Status)
			assert.Equal(t, map[string]string{"database": "ok"}, body.Checks)
		}
	})

	t.Run("is unhealthy when the cache server is down", func(t *testing.T) {
		app := testkit.New(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := health.NewHandler(app.Queries, brokenCache{cache.NewMemoryCache()}, &config.Config{AppName: "TestApp"}, app.Logger)
		router.GET("/health", handler.Check)

		server := httptest.NewServer(router)
		defer server.Close()

		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}
