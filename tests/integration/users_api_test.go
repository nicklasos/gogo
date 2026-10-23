package integration

import (
	"fmt"
	"net/http"
	"testing"

	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/middleware"
	"app/internal/users"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createUserBody(email, role string) string {
	return fmt.Sprintf(`{"email": %q, "name": "New User", "password": "password123", "role": %q}`, email, role)
}

func TestUsersAPI_List(t *testing.T) {
	t.Run("super admin lists users of each role with pagination meta", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())
		app.Create.User(factory.Admin())
		app.Create.User(factory.Admin())
		app.Create.User(factory.Admin())

		resp := app.As(superAdmin).GET("/api/v1/users?role=admin&page=1&page_size=2").AssertOK()

		var response users.PaginatedUsersResponse
		resp.JSON(&response)
		assert.Len(t, response.Data, 2)
		assert.GreaterOrEqual(t, response.Pagination.Total, int64(3))
		assert.Equal(t, int32(2), response.Pagination.PerPage)
		for _, item := range response.Data {
			assert.Equal(t, []string{middleware.RoleAdmin}, item.Roles)
		}

		app.As(superAdmin).GET("/api/v1/users?role=super-admin").AssertOK()
	})

	t.Run("admin lists plain users but not admins or super admins", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.Admin())
		app.Create.User()

		resp := app.As(admin).GET("/api/v1/users").AssertOK()

		var response users.PaginatedUsersResponse
		resp.JSON(&response)
		assert.NotEmpty(t, response.Data)
		for _, item := range response.Data {
			assert.Equal(t, []string{middleware.RoleUser}, item.Roles)
		}

		app.As(admin).GET("/api/v1/users?role=admin").AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		app.As(admin).GET("/api/v1/users?role=super-admin").AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
	})

	t.Run("plain user and anonymous are rejected", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).GET("/api/v1/users").AssertForbidden()
		app.GET("/api/v1/users").AssertUnauthorized()
	})

	t.Run("unknown role is a bad request", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())

		app.As(superAdmin).GET("/api/v1/users?role=owner").AssertBadRequest()
	})
}

func TestUsersAPI_Create(t *testing.T) {
	t.Run("super admin creates a super admin, an admin and a user", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())

		for _, role := range middleware.Roles {
			email := "new-" + role + "@example.com"
			resp := app.As(superAdmin).POST("/api/v1/users", createUserBody(email, role)).AssertOK()

			var response users.UserDataResponse
			resp.JSON(&response)
			assert.Equal(t, email, response.Data.Email)
			assert.Equal(t, []string{role}, response.Data.Roles)
		}
	})

	t.Run("admin creates a user but not an admin or super admin", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.Admin())

		app.As(admin).POST("/api/v1/users", createUserBody("plain@example.com", middleware.RoleUser)).AssertOK()

		app.As(admin).POST("/api/v1/users", createUserBody("a@example.com", middleware.RoleAdmin)).AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		app.As(admin).POST("/api/v1/users", createUserBody("s@example.com", middleware.RoleSuperAdmin)).AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)

		app.AssertDatabaseMissing("users", testkit.Row{"email": "s@example.com"})
	})

	t.Run("plain user cannot create anyone", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).POST("/api/v1/users", createUserBody("x@example.com", middleware.RoleUser)).AssertForbidden()
	})

	t.Run("rejects duplicate email, unknown role and short password", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())

		app.As(superAdmin).POST("/api/v1/users", createUserBody("o@example.com", "owner")).AssertInvalid()

		app.As(superAdmin).POST("/api/v1/users", `{"email": "p@example.com", "name": "P", "password": "short", "role": "user"}`).AssertInvalid()

		// Last on purpose: the unique violation aborts the surrounding test transaction
		app.As(superAdmin).POST("/api/v1/users", createUserBody(superAdmin.Email, middleware.RoleAdmin)).AssertError(http.StatusBadRequest, errs.ErrKeyAuthUserExists)
	})
}

func TestUsersAPI_Update(t *testing.T) {
	t.Run("super admin updates an admin", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())
		target := app.Create.User(factory.Admin())

		resp := app.As(superAdmin).PUT("/api/v1/users/%d", `{"email": "renamed@example.com", "name": "Renamed"}`, target.ID).AssertOK()

		var response users.UserDataResponse
		resp.JSON(&response)
		assert.Equal(t, "renamed@example.com", response.Data.Email)
		assert.Equal(t, "Renamed", response.Data.Name)
		assert.Equal(t, []string{middleware.RoleAdmin}, response.Data.Roles)
	})

	t.Run("admin updates a user but not an admin or super admin", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.Admin())
		plain := app.Create.User()
		otherAdmin := app.Create.User(factory.Admin())
		superAdmin := app.Create.User(factory.SuperAdmin())
		mixed := app.Create.User(factory.WithRoles(middleware.RoleUser, middleware.RoleAdmin))

		body := `{"email": "changed@example.com", "name": "Changed"}`

		app.As(admin).PUT("/api/v1/users/%d", body, plain.ID).AssertOK()

		for _, target := range []*db.User{otherAdmin, superAdmin, mixed} {
			app.As(admin).PUT("/api/v1/users/%d", body, target.ID).AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		}
	})

	t.Run("returns 404 for an unknown user", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())

		app.As(superAdmin).PUT("/api/v1/users/2147483647", `{"email": "x@example.com", "name": "X"}`).AssertError(http.StatusNotFound, errs.ErrKeyUsersNotFound)
	})
}

func TestUsersAPI_SetPassword(t *testing.T) {
	t.Run("sets a new password and revokes refresh tokens", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())
		target := app.Create.User(factory.Admin())

		var session struct {
			Data struct {
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		login(app, target.Email, "password123").JSON(&session)
		require.NotEmpty(t, session.Data.RefreshToken)

		app.As(superAdmin).POST("/api/v1/users/%d/set-password", `{"password": "new-password-1"}`, target.ID).AssertOK()

		login(app, target.Email, "password123").AssertUnauthorized()
		login(app, target.Email, "new-password-1").AssertOK()

		app.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken)).AssertUnauthorized()
	})

	t.Run("admin cannot set the password of an admin", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.Admin())
		target := app.Create.User(factory.SuperAdmin())

		app.As(admin).POST("/api/v1/users/%d/set-password", `{"password": "new-password-1"}`, target.ID).AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
	})
}

func TestUsersAPI_Delete(t *testing.T) {
	t.Run("super admin deletes another super admin but not themselves", func(t *testing.T) {
		app := testkit.New(t)
		superAdmin := app.Create.User(factory.SuperAdmin())
		target := app.Create.User(factory.SuperAdmin())

		app.As(superAdmin).DELETE("/api/v1/users/%d", target.ID).AssertOK()
		app.AssertDatabaseMissing("users", testkit.Row{"id": target.ID})

		app.As(superAdmin).DELETE("/api/v1/users/%d", superAdmin.ID).AssertError(http.StatusBadRequest, errs.ErrKeyUsersCannotDeleteSelf)
	})

	t.Run("admin deletes a user but not an admin", func(t *testing.T) {
		app := testkit.New(t)
		admin := app.Create.User(factory.Admin())
		plain := app.Create.User()
		otherAdmin := app.Create.User(factory.Admin())

		app.As(admin).DELETE("/api/v1/users/%d", plain.ID).AssertOK()

		app.As(admin).DELETE("/api/v1/users/%d", otherAdmin.ID).AssertError(http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		app.AssertDatabaseHas("users", testkit.Row{"id": otherAdmin.ID})
	})
}

func TestUsersAPI_RoleRevocationIsImmediate(t *testing.T) {
	app := testkit.New(t)
	admin := app.Create.User(factory.Admin())
	app.As(admin).GET("/api/v1/users").AssertOK()

	_, err := app.Tx.Exec(app.Ctx, "UPDATE users SET roles = ARRAY['user'] WHERE id = $1", admin.ID)
	require.NoError(t, err)

	app.As(admin).GET("/api/v1/users").AssertForbidden()
}

func TestUsersAPI_DeletedUserTokenIsRejected(t *testing.T) {
	app := testkit.New(t)
	superAdmin := app.Create.User(factory.SuperAdmin())
	target := app.Create.User(factory.Admin())
	app.As(target).GET("/api/v1/auth/me").AssertOK()

	app.As(superAdmin).DELETE("/api/v1/users/%d", target.ID).AssertOK()

	app.As(target).GET("/api/v1/auth/me").AssertError(http.StatusUnauthorized, errs.ErrKeyAuthInvalidToken)
}
