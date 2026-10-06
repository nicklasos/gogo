package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/middleware"
	"app/internal/users"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usersTestActor struct {
	user  *db.User
	token string
}

func newUsersTestActor(t *testing.T, ctx context.Context, tx pgx.Tx, role string) usersTestActor {
	user := factory.User(t, tx, factory.WithRoles(role))
	return usersTestActor{user: user, token: helpers.GenerateTestJWT(user.ID, user.Email)}
}

func createUserBody(email, role string) string {
	return fmt.Sprintf(`{"email": %q, "name": "New User", "password": "password123", "role": %q}`, email, role)
}

func assertErrorKey(t *testing.T, resp *helpers.TestResponse, status int, key string) {
	t.Helper()
	assert.Equal(t, status, resp.StatusCode)
	var body errs.ErrorResponse
	require.NoError(t, resp.JSON(&body))
	assert.Equal(t, key, body.ErrorKey)
}

func TestUsersAPI_List(t *testing.T) {
	t.Run("super admin lists users of each role with pagination meta", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)
			factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))
			factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))
			factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))

			resp := server.GETAuth("/api/v1/users?role=admin&page=1&page_size=2", superAdmin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response users.PaginatedUsersResponse
			require.NoError(t, resp.JSON(&response))
			assert.Len(t, response.Data, 2)
			assert.GreaterOrEqual(t, response.Pagination.Total, int64(3))
			assert.Equal(t, int32(2), response.Pagination.PerPage)
			for _, item := range response.Data {
				assert.Equal(t, []string{middleware.RoleAdmin}, item.Roles)
			}

			resp = server.GETAuth("/api/v1/users?role=super-admin", superAdmin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("admin lists plain users but not admins or super admins", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
			factory.User(t, tx, factory.WithRoles(middleware.RoleUser))

			resp := server.GETAuth("/api/v1/users", admin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response users.PaginatedUsersResponse
			require.NoError(t, resp.JSON(&response))
			assert.NotEmpty(t, response.Data)
			for _, item := range response.Data {
				assert.Equal(t, []string{middleware.RoleUser}, item.Roles)
			}

			assertErrorKey(t, server.GETAuth("/api/v1/users?role=admin", admin.token), http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
			assertErrorKey(t, server.GETAuth("/api/v1/users?role=super-admin", admin.token), http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		})
	})

	t.Run("plain user and anonymous are rejected", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := newUsersTestActor(t, ctx, tx, middleware.RoleUser)

			assert.Equal(t, http.StatusForbidden, server.GETAuth("/api/v1/users", user.token).StatusCode)
			assert.Equal(t, http.StatusUnauthorized, server.GET("/api/v1/users").StatusCode)
		})
	})

	t.Run("unknown role is a bad request", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)

			assert.Equal(t, http.StatusBadRequest, server.GETAuth("/api/v1/users?role=owner", superAdmin.token).StatusCode)
		})
	})
}

func TestUsersAPI_Create(t *testing.T) {
	t.Run("super admin creates a super admin, an admin and a user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)

			for _, role := range middleware.Roles {
				email := "new-" + role + "@example.com"
				resp := server.POSTAuth("/api/v1/users", createUserBody(email, role), superAdmin.token)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response users.UserDataResponse
				require.NoError(t, resp.JSON(&response))
				assert.Equal(t, email, response.Data.Email)
				assert.Equal(t, []string{role}, response.Data.Roles)
			}
		})
	})

	t.Run("admin creates a user but not an admin or super admin", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)

			resp := server.POSTAuth("/api/v1/users", createUserBody("plain@example.com", middleware.RoleUser), admin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assertErrorKey(t, server.POSTAuth("/api/v1/users", createUserBody("a@example.com", middleware.RoleAdmin), admin.token), http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
			assertErrorKey(t, server.POSTAuth("/api/v1/users", createUserBody("s@example.com", middleware.RoleSuperAdmin), admin.token), http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)

			_, err := queries.GetUserByEmail(ctx, "s@example.com")
			assert.ErrorIs(t, err, pgx.ErrNoRows)
		})
	})

	t.Run("plain user cannot create anyone", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := newUsersTestActor(t, ctx, tx, middleware.RoleUser)

			resp := server.POSTAuth("/api/v1/users", createUserBody("x@example.com", middleware.RoleUser), user.token)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	})

	t.Run("rejects duplicate email, unknown role and short password", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)

			assertErrorKey(t, server.POSTAuth("/api/v1/users", createUserBody("o@example.com", "owner"), superAdmin.token), http.StatusBadRequest, errs.ErrKeyValidationFailed)

			resp := server.POSTAuth("/api/v1/users", `{"email": "p@example.com", "name": "P", "password": "short", "role": "user"}`, superAdmin.token)
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyValidationFailed)

			// Last on purpose: the unique violation aborts the surrounding test transaction
			assertErrorKey(t, server.POSTAuth("/api/v1/users", createUserBody(superAdmin.user.Email, middleware.RoleAdmin), superAdmin.token), http.StatusBadRequest, errs.ErrKeyAuthUserExists)
		})
	})
}

func TestUsersAPI_Update(t *testing.T) {
	t.Run("super admin updates an admin", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)
			target := factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))

			resp := server.PUTAuth(fmt.Sprintf("/api/v1/users/%d", target.ID), `{"email": "renamed@example.com", "name": "Renamed"}`, superAdmin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response users.UserDataResponse
			require.NoError(t, resp.JSON(&response))
			assert.Equal(t, "renamed@example.com", response.Data.Email)
			assert.Equal(t, "Renamed", response.Data.Name)
			assert.Equal(t, []string{middleware.RoleAdmin}, response.Data.Roles)
		})
	})

	t.Run("admin updates a user but not an admin or super admin", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
			plain := factory.User(t, tx, factory.WithRoles(middleware.RoleUser))
			otherAdmin := factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))
			superAdmin := factory.User(t, tx, factory.WithRoles(middleware.RoleSuperAdmin))
			mixed := factory.User(t, tx, factory.WithRoles(middleware.RoleUser, middleware.RoleAdmin))

			body := `{"email": "changed@example.com", "name": "Changed"}`

			assert.Equal(t, http.StatusOK, server.PUTAuth(fmt.Sprintf("/api/v1/users/%d", plain.ID), body, admin.token).StatusCode)

			for _, target := range []*db.User{otherAdmin, superAdmin, mixed} {
				resp := server.PUTAuth(fmt.Sprintf("/api/v1/users/%d", target.ID), body, admin.token)
				assertErrorKey(t, resp, http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
			}
		})
	})

	t.Run("returns 404 for an unknown user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)

			resp := server.PUTAuth("/api/v1/users/2147483647", `{"email": "x@example.com", "name": "X"}`, superAdmin.token)
			assertErrorKey(t, resp, http.StatusNotFound, errs.ErrKeyUsersNotFound)
		})
	})
}

func TestUsersAPI_SetPassword(t *testing.T) {
	t.Run("sets a new password and revokes refresh tokens", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)
			target := factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))

			login := func(password string) *helpers.TestResponse {
				return server.POST("/api/v1/auth/login", fmt.Sprintf(`{"email": %q, "password": %q}`, target.Email, password))
			}

			var session struct {
				Data struct {
					RefreshToken string `json:"refresh_token"`
				} `json:"data"`
			}
			require.NoError(t, login("password123").JSON(&session))
			require.NotEmpty(t, session.Data.RefreshToken)

			resp := server.POSTAuth(fmt.Sprintf("/api/v1/users/%d/set-password", target.ID), `{"password": "new-password-1"}`, superAdmin.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, http.StatusUnauthorized, login("password123").StatusCode)
			assert.Equal(t, http.StatusOK, login("new-password-1").StatusCode)

			refresh := server.POST("/api/v1/auth/refresh", fmt.Sprintf(`{"refresh_token": %q}`, session.Data.RefreshToken))
			assert.Equal(t, http.StatusUnauthorized, refresh.StatusCode)
		})
	})

	t.Run("admin cannot set the password of an admin", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
			target := factory.User(t, tx, factory.WithRoles(middleware.RoleSuperAdmin))

			resp := server.POSTAuth(fmt.Sprintf("/api/v1/users/%d/set-password", target.ID), `{"password": "new-password-1"}`, admin.token)
			assertErrorKey(t, resp, http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
		})
	})
}

func TestUsersAPI_Delete(t *testing.T) {
	t.Run("super admin deletes another super admin but not themselves", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)
			target := factory.User(t, tx, factory.WithRoles(middleware.RoleSuperAdmin))

			assert.Equal(t, http.StatusOK, server.DELETEAuth(fmt.Sprintf("/api/v1/users/%d", target.ID), superAdmin.token).StatusCode)
			_, err := queries.GetUserByID(ctx, target.ID)
			assert.ErrorIs(t, err, pgx.ErrNoRows)

			resp := server.DELETEAuth(fmt.Sprintf("/api/v1/users/%d", superAdmin.user.ID), superAdmin.token)
			assertErrorKey(t, resp, http.StatusBadRequest, errs.ErrKeyUsersCannotDeleteSelf)
		})
	})

	t.Run("admin deletes a user but not an admin", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
			plain := factory.User(t, tx, factory.WithRoles(middleware.RoleUser))
			otherAdmin := factory.User(t, tx, factory.WithRoles(middleware.RoleAdmin))

			assert.Equal(t, http.StatusOK, server.DELETEAuth(fmt.Sprintf("/api/v1/users/%d", plain.ID), admin.token).StatusCode)

			resp := server.DELETEAuth(fmt.Sprintf("/api/v1/users/%d", otherAdmin.ID), admin.token)
			assertErrorKey(t, resp, http.StatusForbidden, errs.ErrKeyUsersForbiddenRole)
			_, err := queries.GetUserByID(ctx, otherAdmin.ID)
			assert.NoError(t, err)
		})
	})
}

func TestUsersAPI_RoleRevocationIsImmediate(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		admin := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
		assert.Equal(t, http.StatusOK, server.GETAuth("/api/v1/users", admin.token).StatusCode)

		_, err := tx.Exec(ctx, "UPDATE users SET roles = ARRAY['user'] WHERE id = $1", admin.user.ID)
		require.NoError(t, err)

		assert.Equal(t, http.StatusForbidden, server.GETAuth("/api/v1/users", admin.token).StatusCode)
	})
}

func TestUsersAPI_DeletedUserTokenIsRejected(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		superAdmin := newUsersTestActor(t, ctx, tx, middleware.RoleSuperAdmin)
		target := newUsersTestActor(t, ctx, tx, middleware.RoleAdmin)
		assert.Equal(t, http.StatusOK, server.GETAuth("/api/v1/auth/me", target.token).StatusCode)

		require.Equal(t, http.StatusOK, server.DELETEAuth(fmt.Sprintf("/api/v1/users/%d", target.user.ID), superAdmin.token).StatusCode)

		assertErrorKey(t, server.GETAuth("/api/v1/auth/me", target.token), http.StatusUnauthorized, errs.ErrKeyAuthInvalidToken)
	})
}
