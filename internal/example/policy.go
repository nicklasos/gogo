package example

import (
	"app/internal/db"
	"app/internal/middleware"
)

// Who may do what to an example. The service checks these after loading the record,
// and gogo-front mirrors them in features/examples/policy.ts to hide what is not allowed.

func CanView(actor middleware.Actor, example db.Example) bool {
	return example.UserID == actor.ID || actor.IsAdmin()
}

// CanUpdate is the owner only: an admin can look at and remove someone's example, not rewrite it.
func CanUpdate(actor middleware.Actor, example db.Example) bool {
	return example.UserID == actor.ID
}

func CanDelete(actor middleware.Actor, example db.Example) bool {
	return example.UserID == actor.ID || actor.IsAdmin()
}
