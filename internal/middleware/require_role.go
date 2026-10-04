package middleware

import (
	"context"
	"slices"

	"app/internal/errs"

	"github.com/gin-gonic/gin"
)

const (
	RoleSuperAdmin = "super-admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

var Roles = []string{RoleSuperAdmin, RoleAdmin, RoleUser}

// UserRolesLookup loads the current roles of the authenticated user.
type UserRolesLookup interface {
	GetUserRoles(ctx context.Context, userID int32) ([]string, error)
}

// UserAuth is JWT verification plus roles lookup. AuthService implements this.
type UserAuth interface {
	UserJWTVerifier
	UserRolesLookup
}

// HasAnyRole reports whether the roles satisfy a requirement. A super admin satisfies every requirement.
func HasAnyRole(userRoles []string, required ...string) bool {
	if slices.Contains(userRoles, RoleSuperAdmin) {
		return true
	}
	for _, role := range required {
		if slices.Contains(userRoles, role) {
			return true
		}
	}
	return false
}

// RequireRole requires a role already loaded onto the context by UserAuthMiddleware.
func RequireRole(role string) gin.HandlerFunc {
	return RequireAnyRole(role)
}

// RequireAnyRole requires at least one of the given roles already loaded onto the context.
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user_roles"); !exists {
			errs.RespondWithUnauthorized(c, "Unauthorized")
			c.Abort()
			return
		}

		if !HasAnyRole(GetUserRolesFromContext(c), roles...) {
			errs.RespondWithError(c, errs.NewForbiddenError(errs.ErrKeyForbidden, "Forbidden"))
			c.Abort()
			return
		}

		c.Next()
	}
}
