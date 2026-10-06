package integration

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"app/config"
	"app/internal/auth"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/mail"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emailTokenPattern = regexp.MustCompile(`token=([0-9a-f]+)`)

func emailToken(t *testing.T, msg mail.Message) string {
	t.Helper()
	match := emailTokenPattern.FindStringSubmatch(msg.Text)
	require.Len(t, match, 2, "no token link in the email text")
	assert.Contains(t, msg.HTML, match[1], "the HTML body must carry the same link")
	return match[1]
}

func loginStatus(server *helpers.TestServer, email, password string) int {
	return server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": %q}`, email, password)).StatusCode
}

func TestAuthAPI_PasswordReset(t *testing.T) {
	t.Run("emails a link that sets a new password exactly once", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx, factory.Unverified())

			var session auth.LoginDataResponse
			resp := server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
			require.NoError(t, resp.JSON(&session))

			resp = server.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email))
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			sent := server.Mail.Sent()
			require.Len(t, sent, 1)
			assert.Equal(t, []string{user.Email}, sent[0].To)
			assert.Contains(t, sent[0].Text, "http://localhost:5173/reset-password?token=")
			token := emailToken(t, sent[0])

			resetBody := fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, token)
			assert.Equal(t, http.StatusOK, server.POST("/api/v1/auth/reset-password", resetBody).StatusCode)

			assert.Equal(t, http.StatusUnauthorized, loginStatus(server, user.Email, "password123"))
			assert.Equal(t, http.StatusOK, loginStatus(server, user.Email, "brand-new-password"))

			refresh := server.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken))
			assert.Equal(t, http.StatusUnauthorized, refresh.StatusCode, "sessions opened before the reset must not be renewable")

			assertErrorKey(t, server.POST("/api/v1/auth/reset-password", resetBody), http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

			updated, err := queries.GetUserByID(ctx, user.ID)
			require.NoError(t, err)
			assert.True(t, updated.EmailVerifiedAt.Valid, "opening the emailed link proves the address")
		})
	})

	t.Run("answers 200 and sends nothing for an unknown email", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			resp := server.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Empty(t, server.Mail.Sent())
		})
	})

	t.Run("only the newest link works", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			body := fmt.Sprintf(`{"email": %q}`, user.Email)
			server.POST("/api/v1/auth/forgot-password", body)
			server.POST("/api/v1/auth/forgot-password", body)

			sent := server.Mail.Sent()
			require.Len(t, sent, 2)
			first, second := emailToken(t, sent[0]), emailToken(t, sent[1])
			require.NotEqual(t, first, second)

			resp := server.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, first))
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

			resp = server.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, second))
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("rejects an expired link, a verification token and a short password", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx, factory.Unverified())
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			server.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email))
			resetToken := emailToken(t, server.Mail.Sent()[0])

			resp := server.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "short"}`, resetToken))
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyValidationFailed)

			require.Equal(t, http.StatusOK, server.POSTAuth("/api/v1/auth/me/verify-email", nil, token).StatusCode)
			verifyToken := emailToken(t, server.Mail.Sent()[1])
			resp = server.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, verifyToken))
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

			_, err := tx.Exec(ctx, "UPDATE auth_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 minute' WHERE user_id = $1", user.ID)
			require.NoError(t, err)
			resp = server.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, resetToken))
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

			assert.Equal(t, http.StatusOK, loginStatus(server, user.Email, "password123"))
		})
	})

	t.Run("stores only a hash of the token", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			server.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email))
			token := emailToken(t, server.Mail.Sent()[0])

			var stored string
			require.NoError(t, tx.QueryRow(ctx, "SELECT token_hash FROM auth_tokens WHERE user_id = $1", user.ID).Scan(&stored))
			assert.NotEqual(t, token, stored)
			assert.Len(t, stored, 64)
		})
	})
}

func TestAuthAPI_EmailVerification(t *testing.T) {
	t.Run("registration sends a link that verifies the email", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			var registered auth.RegisterDataResponse
			resp := server.POST("/api/v1/auth/register", `{"email": "fresh@example.com", "name": "Fresh", "password": "password123"}`)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NoError(t, resp.JSON(&registered))
			assert.False(t, registered.Data.User.EmailVerified)

			sent := server.Mail.Sent()
			require.Len(t, sent, 1)
			assert.Contains(t, sent[0].Text, "http://localhost:5173/verify-email?token=")
			body := fmt.Sprintf(`{"token": %q}`, emailToken(t, sent[0]))

			assert.Equal(t, http.StatusOK, server.POST("/api/v1/auth/verify-email", body).StatusCode)

			var me auth.UserDataResponse
			require.NoError(t, server.GETAuth("/api/v1/auth/me", registered.Data.AccessToken).JSON(&me))
			assert.True(t, me.Data.EmailVerified)

			assertErrorKey(t, server.POST("/api/v1/auth/verify-email", body), http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)
		})
	})

	t.Run("resend works until verified, then is refused", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx, factory.Unverified())
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			require.Equal(t, http.StatusOK, server.POSTAuth("/api/v1/auth/me/verify-email", nil, token).StatusCode)
			link := emailToken(t, server.Mail.Sent()[0])
			require.Equal(t, http.StatusOK, server.POST("/api/v1/auth/verify-email", fmt.Sprintf(`{"token": %q}`, link)).StatusCode)

			assertErrorKey(t, server.POSTAuth("/api/v1/auth/me/verify-email", nil, token), http.StatusBadRequest, errs.ErrKeyAuthEmailVerified)
			assert.Equal(t, http.StatusUnauthorized, server.POST("/api/v1/auth/me/verify-email", nil).StatusCode)
		})
	})

	t.Run("changing the email makes it unverified and sends a new link", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			require.NoError(t, queries.MarkUserEmailVerified(ctx, user.ID))
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			var updated auth.UserDataResponse
			resp := server.PUTAuth("/api/v1/auth/me", fmt.Sprintf(`{"email": %q, "name": "Renamed"}`, user.Email), token)
			require.NoError(t, resp.JSON(&updated))
			assert.True(t, updated.Data.EmailVerified, "a rename keeps the email verified")
			assert.Empty(t, server.Mail.Sent())

			resp = server.PUTAuth("/api/v1/auth/me", `{"email": "moved@example.com", "name": "Renamed"}`, token)
			require.NoError(t, resp.JSON(&updated))
			assert.False(t, updated.Data.EmailVerified)

			sent := server.Mail.Sent()
			require.Len(t, sent, 1)
			assert.Equal(t, []string{"moved@example.com"}, sent[0].To)
		})
	})

	t.Run("accounts created by an admin are verified and stay so when the admin edits them", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := factory.User(t, tx, factory.WithRoles("super-admin"))
			token := helpers.GenerateTestJWT(admin.ID, admin.Email)

			var created struct {
				Data struct {
					ID            int32 `json:"id"`
					EmailVerified bool  `json:"email_verified"`
				} `json:"data"`
			}
			resp := server.POSTAuth("/api/v1/users", `{"email": "made@example.com", "name": "Made", "password": "password123", "role": "user"}`, token)
			require.NoError(t, resp.JSON(&created))
			assert.True(t, created.Data.EmailVerified)

			resp = server.PUTAuth(fmt.Sprintf("/api/v1/users/%d", created.Data.ID), `{"email": "made-2@example.com", "name": "Made"}`, token)
			require.NoError(t, resp.JSON(&created))
			assert.True(t, created.Data.EmailVerified)
			assert.Empty(t, server.Mail.Sent())
		})
	})
}

func TestAuthAPI_RegistrationDisabled(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries, func(cfg *config.Config) {
			cfg.AllowRegistration = false
		})
		defer server.Close()

		resp := server.POST("/api/v1/auth/register", `{"email": "closed@example.com", "name": "Closed", "password": "password123"}`)
		assertErrorKey(t, resp, http.StatusForbidden, errs.ErrKeyAuthRegistrationClosed)

		_, err := queries.GetUserByEmail(ctx, "closed@example.com")
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})
}

func TestAuthAPI_LoginRateLimit(t *testing.T) {
	t.Run("blocks an email after repeated failures and says how long to wait", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			other := factory.User(t, tx)

			for i := 0; i < 40; i++ {
				require.Equal(t, http.StatusUnauthorized, loginStatus(server, "target@example.com", "wrong"), "attempt %d", i+1)
			}

			resp := server.POST("/api/v1/auth/login", `{"email": "target@example.com", "password": "wrong"}`)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			assert.NotEmpty(t, resp.Header.Get("Retry-After"))

			var body errs.ErrorResponse
			require.NoError(t, resp.JSON(&body))
			assert.Equal(t, errs.ErrKeyAuthTooManyRequests, body.ErrorKey)
			assert.Greater(t, body.Details["retry_after_seconds"], float64(0))

			assert.Equal(t, http.StatusOK, loginStatus(server, other.Email, "password123"), "another account from the same address is not locked out")
		})
	})

	t.Run("a successful login clears the failure count", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx, factory.WithRoles("user"))

			for round := 0; round < 2; round++ {
				for i := 0; i < 25; i++ {
					require.Equal(t, http.StatusUnauthorized, loginStatus(server, user.Email, "wrong"))
				}
				require.Equal(t, http.StatusOK, loginStatus(server, user.Email, "password123"))
			}
		})
	})

	t.Run("can be switched off", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries, func(cfg *config.Config) {
				cfg.AuthRateLimit = false
			})
			defer server.Close()

			for i := 0; i < 45; i++ {
				require.Equal(t, http.StatusUnauthorized, loginStatus(server, "target@example.com", "wrong"))
			}
		})
	})

	t.Run("limits how many reset emails one address can trigger", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			for i := 0; i < 10; i++ {
				require.Equal(t, http.StatusOK, server.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`).StatusCode)
			}
			resp := server.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		})
	})
}
