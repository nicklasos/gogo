package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"app/internal/db"
	"app/internal/example"
	"app/internal/middleware"
	"app/internal/users"
)

func TestActor(t *testing.T) {
	user := middleware.Actor{ID: 1, Roles: []string{middleware.RoleUser}}
	admin := middleware.Actor{ID: 2, Roles: []string{middleware.RoleAdmin}}
	superAdmin := middleware.Actor{ID: 3, Roles: []string{middleware.RoleSuperAdmin}}

	assert.False(t, user.IsAdmin())
	assert.True(t, admin.IsAdmin())
	assert.True(t, superAdmin.IsAdmin(), "a super admin passes every admin check")
	assert.False(t, admin.IsSuperAdmin())
	assert.True(t, superAdmin.IsSuperAdmin())
	assert.False(t, middleware.Actor{}.IsAdmin())
}

func TestExamplePolicy(t *testing.T) {
	owner := middleware.Actor{ID: 1, Roles: []string{middleware.RoleUser}}
	stranger := middleware.Actor{ID: 2, Roles: []string{middleware.RoleUser}}
	admin := middleware.Actor{ID: 3, Roles: []string{middleware.RoleAdmin}}
	superAdmin := middleware.Actor{ID: 4, Roles: []string{middleware.RoleSuperAdmin}}
	record := db.Example{ID: 10, UserID: owner.ID}

	cases := []struct {
		name                string
		actor               middleware.Actor
		view, edit, destroy bool
	}{
		{"owner", owner, true, true, true},
		{"another user", stranger, false, false, false},
		{"admin", admin, true, false, true},
		{"super admin", superAdmin, true, false, true},
		{"nobody", middleware.Actor{}, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.view, example.CanView(tc.actor, record), "view")
			assert.Equal(t, tc.edit, example.CanUpdate(tc.actor, record), "update")
			assert.Equal(t, tc.destroy, example.CanDelete(tc.actor, record), "delete")
		})
	}
}

func TestUsersPolicy_CanManage(t *testing.T) {
	user := middleware.Actor{Roles: []string{middleware.RoleUser}}
	admin := middleware.Actor{Roles: []string{middleware.RoleAdmin}}
	superAdmin := middleware.Actor{Roles: []string{middleware.RoleSuperAdmin}}

	cases := []struct {
		name   string
		actor  middleware.Actor
		target []string
		want   bool
	}{
		{"super admin manages a super admin", superAdmin, []string{middleware.RoleSuperAdmin}, true},
		{"super admin manages an admin", superAdmin, []string{middleware.RoleAdmin}, true},
		{"super admin manages a user", superAdmin, []string{middleware.RoleUser}, true},
		{"admin manages a user", admin, []string{middleware.RoleUser}, true},
		{"admin does not manage an admin", admin, []string{middleware.RoleAdmin}, false},
		{"admin does not manage a super admin", admin, []string{middleware.RoleSuperAdmin}, false},
		{"admin does not manage a user who is also an admin", admin, []string{middleware.RoleUser, middleware.RoleAdmin}, false},
		{"admin does not manage an account without roles", admin, nil, false},
		{"user manages nobody", user, []string{middleware.RoleUser}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, users.CanManage(tc.actor, tc.target))
		})
	}
}
