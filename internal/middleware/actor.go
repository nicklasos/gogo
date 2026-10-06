package middleware

import (
	"slices"

	"github.com/gin-gonic/gin"
)

// Actor is who is making the request: the input of every policy function.
// Handlers get it from CurrentActor and pass it to the service.
type Actor struct {
	ID    int32
	Roles []string
}

func (a Actor) IsSuperAdmin() bool {
	return slices.Contains(a.Roles, RoleSuperAdmin)
}

// IsAdmin is true for admins and for super admins.
func (a Actor) IsAdmin() bool {
	return HasAnyRole(a.Roles, RoleAdmin)
}

// CurrentActor returns the authenticated user and their roles. When there is none it
// answers 401 itself and returns false, so a handler only has to return.
func CurrentActor(c *gin.Context) (Actor, bool) {
	userID, ok := CurrentUserID(c)
	if !ok {
		return Actor{}, false
	}
	return Actor{ID: userID, Roles: GetUserRolesFromContext(c)}, true
}
