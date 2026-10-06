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
	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/health"
	"app/tests/helpers"

	"app/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "user@example.com", internal.NormalizeEmail("  User@Example.COM "))
	assert.Equal(t, "", internal.NormalizeEmail("   "))
}

func TestAuthAPI_EmailsAreCaseInsensitive(t *testing.T) {
	t.Run("login, reset and duplicates ignore case", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			var registered auth.RegisterDataResponse
			resp := server.POST("/api/v1/auth/register", `{"email": "Mixed.Case@Example.com", "name": "Mixed", "password": "password123"}`)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NoError(t, resp.JSON(&registered))
			assert.Equal(t, "mixed.case@example.com", registered.Data.User.Email, "stored lower-case")

			for _, email := range []string{"mixed.case@example.com", "MIXED.CASE@EXAMPLE.COM", "Mixed.Case@Example.com"} {
				assert.Equal(t, http.StatusOK, loginStatus(server, email, "password123"), email)
			}

			sentBefore := len(server.Mail.Sent())
			server.POST("/api/v1/auth/forgot-password", `{"email": "MIXED.CASE@example.com"}`)
			require.Len(t, server.Mail.Sent(), sentBefore+1)
			assert.Equal(t, []string{"mixed.case@example.com"}, server.Mail.Sent()[sentBefore].To)

			resp = server.POST("/api/v1/auth/register", `{"email": "MIXED.case@example.com", "name": "Again", "password": "password123"}`)
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyAuthUserExists)
		})
	})

	t.Run("an account stored with capitals before the change can still sign in", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			factory.User(t, tx, factory.WithEmail("Legacy.User@example.com"))

			assert.Equal(t, http.StatusOK, loginStatus(server, "legacy.user@example.com", "password123"))
			assert.Equal(t, http.StatusOK, loginStatus(server, "Legacy.User@example.com", "password123"))
		})
	})

	t.Run("the database refuses two accounts that differ only by case", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			factory.User(t, tx, factory.WithEmail("twin@example.com"))

			_, err := tx.Exec(ctx, "INSERT INTO users (email, name, password) VALUES ('TWIN@example.com', 'Twin', 'x')")
			require.Error(t, err)
			assert.NotNil(t, errs.DomainErrorFromPostgresUniqueViolation(err), "mapped to the user-exists error")
		})
	})

	t.Run("admins create and edit users with normalised emails", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := factory.User(t, tx, factory.WithRoles("super-admin"))
			token := helpers.GenerateTestJWT(admin.ID, admin.Email)

			var created struct {
				Data struct {
					ID    int32  `json:"id"`
					Email string `json:"email"`
				} `json:"data"`
			}
			resp := server.POSTAuth("/api/v1/users", `{"email": "New.Person@Example.com", "name": "New", "password": "password123", "role": "user"}`, token)
			require.NoError(t, resp.JSON(&created))
			assert.Equal(t, "new.person@example.com", created.Data.Email)

			resp = server.PUTAuth(fmt.Sprintf("/api/v1/users/%d", created.Data.ID), `{"email": "Renamed.Person@Example.com", "name": "New"}`, token)
			require.NoError(t, resp.JSON(&created))
			assert.Equal(t, "renamed.person@example.com", created.Data.Email)
		})
	})
}

func TestAuthAPI_RefreshTokensAreStoredHashed(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := factory.User(t, tx)

		var session auth.LoginDataResponse
		resp := server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
		require.NoError(t, resp.JSON(&session))
		raw := session.Data.RefreshToken
		require.NotEmpty(t, raw)

		var stored string
		require.NoError(t, tx.QueryRow(ctx, "SELECT token FROM refresh_tokens WHERE user_id = $1", user.ID).Scan(&stored))
		sum := sha256.Sum256([]byte(raw))
		assert.Equal(t, hex.EncodeToString(sum[:]), stored)
		assert.NotEqual(t, raw, stored)

		refresh := func(token string) int {
			return server.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, token)).StatusCode
		}
		assert.Equal(t, http.StatusUnauthorized, refresh(stored), "the stored value is not a usable token")
		assert.Equal(t, http.StatusOK, refresh(raw))
		assert.Equal(t, http.StatusUnauthorized, refresh(raw), "still single-use")
	})
}

func TestAuth_TokenInQueryStringIsIgnored(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := factory.User(t, tx)
		token := helpers.GenerateTestJWT(user.ID, user.Email)

		assert.Equal(t, http.StatusUnauthorized, server.GET("/api/v1/auth/me?token="+token).StatusCode)
		assert.Equal(t, http.StatusOK, server.GETAuth("/api/v1/auth/me", token).StatusCode)
	})
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
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			for _, path := range []string{"/health", "/api/v1/health"} {
				resp := server.GET(path)
				assert.Equal(t, http.StatusOK, resp.StatusCode, path)

				var body healthBody
				require.NoError(t, resp.JSON(&body))
				assert.Equal(t, "healthy", body.Status)
				assert.Equal(t, map[string]string{"database": "ok"}, body.Checks)
			}
		})
	})

	t.Run("is unhealthy when the cache server is down", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			handler := health.NewHandler(queries, brokenCache{cache.NewMemoryCache()}, &config.Config{AppName: "TestApp"}, helpers.GetTestLogger(t))
			router.GET("/health", handler.Check)

			server := httptest.NewServer(router)
			defer server.Close()

			resp, err := http.Get(server.URL + "/health")
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	})
}
