package integration

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"app/config"
	"app/internal/auth"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/mail"
	"app/tests/testkit"

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

func login(app *testkit.App, email, password string) *testkit.Response {
	return app.POST("/api/v1/auth/login", auth.LoginRequest{Email: email, Password: password})
}

func TestAuthAPI_PasswordReset(t *testing.T) {
	t.Run("emails a link that sets a new password exactly once", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User(factory.Unverified())

		var session auth.LoginDataResponse
		resp := app.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": "password123"}`, user.Email))
		resp.JSON(&session)

		resp = app.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email)).AssertOK()

		sent := app.Mail.Sent()
		require.Len(t, sent, 1)
		assert.Equal(t, []string{user.Email}, sent[0].To)
		assert.Contains(t, sent[0].Text, "http://localhost:5173/reset-password?token=")
		token := emailToken(t, sent[0])

		resetBody := fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, token)
		app.POST("/api/v1/auth/reset-password", resetBody).AssertOK()

		login(app, user.Email, "password123").AssertUnauthorized()
		login(app, user.Email, "brand-new-password").AssertOK()

		refresh := app.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken))
		assert.Equal(t, http.StatusUnauthorized, refresh.StatusCode, "sessions opened before the reset must not be renewable")

		app.POST("/api/v1/auth/reset-password", resetBody).AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

		updated, err := app.Queries.GetUserByID(app.Ctx, user.ID)
		require.NoError(t, err)
		assert.True(t, updated.EmailVerifiedAt.Valid, "opening the emailed link proves the address")
	})

	t.Run("answers 200 and sends nothing for an unknown email", func(t *testing.T) {
		app := testkit.New(t)
		app.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`).AssertOK()
		assert.Empty(t, app.Mail.Sent())
	})

	t.Run("only the newest link works", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		body := fmt.Sprintf(`{"email": %q}`, user.Email)
		app.POST("/api/v1/auth/forgot-password", body)
		app.POST("/api/v1/auth/forgot-password", body)

		sent := app.Mail.Sent()
		require.Len(t, sent, 2)
		first, second := emailToken(t, sent[0]), emailToken(t, sent[1])
		require.NotEqual(t, first, second)

		app.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, first)).AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

		app.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, second)).AssertOK()
	})

	t.Run("rejects an expired link, a verification token and a short password", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User(factory.Unverified())
		token := app.Token(user)

		app.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email))
		resetToken := emailToken(t, app.Mail.Sent()[0])

		app.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "short"}`, resetToken)).AssertError(http.StatusBadRequest, errs.ErrKeyValidationFailed)

		app.WithToken(token).POST("/api/v1/auth/me/verify-email", nil).AssertOK()
		verifyToken := emailToken(t, app.Mail.Sent()[1])
		app.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, verifyToken)).AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

		_, err := app.Tx.Exec(app.Ctx, "UPDATE auth_tokens SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 minute' WHERE user_id = $1", user.ID)
		require.NoError(t, err)
		app.POST("/api/v1/auth/reset-password", fmt.Sprintf(`{"token": %q, "password": "brand-new-password"}`, resetToken)).AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)

		login(app, user.Email, "password123").AssertOK()
	})

	t.Run("stores only a hash of the token", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		app.POST("/api/v1/auth/forgot-password", fmt.Sprintf(`{"email": %q}`, user.Email))
		token := emailToken(t, app.Mail.Sent()[0])

		var stored string
		require.NoError(t, app.Tx.QueryRow(app.Ctx, "SELECT token_hash FROM auth_tokens WHERE user_id = $1", user.ID).Scan(&stored))
		assert.NotEqual(t, token, stored)
		assert.Len(t, stored, 64)
	})
}

func TestAuthAPI_EmailVerification(t *testing.T) {
	t.Run("registration sends a link that verifies the email", func(t *testing.T) {
		app := testkit.New(t)
		var registered auth.RegisterDataResponse
		resp := app.POST("/api/v1/auth/register", `{"email": "fresh@example.com", "name": "Fresh", "password": "password123"}`).AssertOK()
		resp.JSON(&registered)
		assert.False(t, registered.Data.User.EmailVerified)

		sent := app.Mail.Sent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0].Text, "http://localhost:5173/verify-email?token=")
		body := fmt.Sprintf(`{"token": %q}`, emailToken(t, sent[0]))

		app.POST("/api/v1/auth/verify-email", body).AssertOK()

		var me auth.UserDataResponse
		app.WithToken(registered.Data.AccessToken).GET("/api/v1/auth/me").JSON(&me)
		assert.True(t, me.Data.EmailVerified)

		app.POST("/api/v1/auth/verify-email", body).AssertError(http.StatusBadRequest, errs.ErrKeyAuthInvalidEmailToken)
	})

	t.Run("resend works until verified, then is refused", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User(factory.Unverified())
		token := app.Token(user)

		app.WithToken(token).POST("/api/v1/auth/me/verify-email", nil).AssertOK()
		link := emailToken(t, app.Mail.Sent()[0])
		app.POST("/api/v1/auth/verify-email", fmt.Sprintf(`{"token": %q}`, link)).AssertOK()

		app.WithToken(token).POST("/api/v1/auth/me/verify-email", nil).AssertError(http.StatusBadRequest, errs.ErrKeyAuthEmailVerified)
		app.POST("/api/v1/auth/me/verify-email", nil).AssertUnauthorized()
	})

	t.Run("changing the email makes it unverified and sends a new link", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		require.NoError(t, app.Queries.MarkUserEmailVerified(app.Ctx, user.ID))
		token := app.Token(user)

		var updated auth.UserDataResponse
		resp := app.WithToken(token).PUT("/api/v1/auth/me", fmt.Sprintf(`{"email": %q, "name": "Renamed"}`, user.Email))
		resp.JSON(&updated)
		assert.True(t, updated.Data.EmailVerified, "a rename keeps the email verified")
		assert.Empty(t, app.Mail.Sent())

		resp = app.WithToken(token).PUT("/api/v1/auth/me", `{"email": "moved@example.com", "name": "Renamed"}`)
		resp.JSON(&updated)
		assert.False(t, updated.Data.EmailVerified)

		sent := app.Mail.Sent()
		require.Len(t, sent, 1)
		assert.Equal(t, []string{"moved@example.com"}, sent[0].To)
	})

	t.Run("accounts created by an admin are verified and stay so when the admin edits them", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.WithRoles("super-admin"))
		token := app.Token(admin)

		var created struct {
			Data struct {
				ID            int32 `json:"id"`
				EmailVerified bool  `json:"email_verified"`
			} `json:"data"`
		}
		resp := app.WithToken(token).POST("/api/v1/users", `{"email": "made@example.com", "name": "Made", "password": "password123", "role": "user"}`)
		resp.JSON(&created)
		assert.True(t, created.Data.EmailVerified)

		resp = app.WithToken(token).PUT("/api/v1/users/%d", `{"email": "made-2@example.com", "name": "Made"}`, created.Data.ID)
		resp.JSON(&created)
		assert.True(t, created.Data.EmailVerified)
		assert.Empty(t, app.Mail.Sent())
	})
}

func TestAuthAPI_RegistrationDisabled(t *testing.T) {
	app := testkit.New(t, func(cfg *config.Config) {
		cfg.AllowRegistration = false
	})

	app.POST("/api/v1/auth/register", `{"email": "closed@example.com", "name": "Closed", "password": "password123"}`).AssertError(http.StatusForbidden, errs.ErrKeyAuthRegistrationClosed)

	_, err := app.Queries.GetUserByEmail(app.Ctx, "closed@example.com")
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestAuthAPI_LoginRateLimit(t *testing.T) {
	t.Run("blocks an email after repeated failures and says how long to wait", func(t *testing.T) {
		app := testkit.New(t)
		other := app.Create.User()

		for i := 0; i < 40; i++ {
			login(app, "target@example.com", "wrong").AssertUnauthorized()
		}

		resp := app.POST("/api/v1/auth/login", `{"email": "target@example.com", "password": "wrong"}`).AssertTooManyRequests()
		assert.NotEmpty(t, resp.Header.Get("Retry-After"))

		var body errs.ErrorResponse
		resp.JSON(&body)
		assert.Equal(t, errs.ErrKeyAuthTooManyRequests, body.ErrorKey)
		assert.Greater(t, body.Details["retry_after_seconds"], float64(0))

		login(app, other.Email, "password123").AssertOK()
	})

	t.Run("a successful login clears the failure count", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User(factory.WithRoles("user"))

		for round := 0; round < 2; round++ {
			for i := 0; i < 25; i++ {
				login(app, user.Email, "wrong").AssertUnauthorized()
			}
			login(app, user.Email, "password123").AssertOK()
		}
	})

	t.Run("can be switched off", func(t *testing.T) {
		app := testkit.New(t, func(cfg *config.Config) {
			cfg.AuthRateLimit = false
		})

		for i := 0; i < 45; i++ {
			login(app, "target@example.com", "wrong").AssertUnauthorized()
		}
	})

	t.Run("limits how many reset emails one address can trigger", func(t *testing.T) {
		app := testkit.New(t)
		for i := 0; i < 10; i++ {
			app.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`).AssertOK()
		}
		app.POST("/api/v1/auth/forgot-password", `{"email": "nobody@example.com"}`).AssertTooManyRequests()
	})
}
