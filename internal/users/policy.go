package users

import "app/internal/middleware"

// CanManage is the single rule for who may list, create, edit and delete whom:
// a super admin manages everyone, an admin manages only accounts whose every role is "user".
// gogo-front mirrors it in features/users/policy.ts.
func CanManage(actor middleware.Actor, targetRoles []string) bool {
	if actor.IsSuperAdmin() {
		return true
	}
	if !actor.IsAdmin() || len(targetRoles) == 0 {
		return false
	}
	for _, role := range targetRoles {
		if role != middleware.RoleUser {
			return false
		}
	}
	return true
}
