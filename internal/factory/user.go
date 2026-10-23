package factory

import (
	"app/internal/db"
	"app/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

type userAttrs struct {
	email    string
	name     string
	password string
	roles    []string
	verified bool
}

type UserOption func(*userAttrs)

func WithEmail(email string) UserOption       { return func(a *userAttrs) { a.email = email } }
func WithName(name string) UserOption         { return func(a *userAttrs) { a.name = name } }
func WithPassword(password string) UserOption { return func(a *userAttrs) { a.password = password } }
func WithRoles(roles ...string) UserOption    { return func(a *userAttrs) { a.roles = roles } }

// Admin and SuperAdmin are the role states: create.User(factory.Admin()).
func Admin() UserOption      { return WithRoles(middleware.RoleAdmin) }
func SuperAdmin() UserOption { return WithRoles(middleware.RoleSuperAdmin) }

// Unverified leaves the email unconfirmed, as it is right after self-registration.
func Unverified() UserOption { return func(a *userAttrs) { a.verified = false } }

// User creates a user with the role "user", a verified email and DefaultPassword.
func (f *Factory) User(opts ...UserOption) *db.User {
	f.t.Helper()

	attrs := userAttrs{
		email:    "user-" + unique() + "@example.com",
		name:     "Test User",
		password: DefaultPassword,
		roles:    []string{"user"},
		verified: true,
	}
	for _, opt := range opts {
		opt(&attrs)
	}

	// The cheapest cost: these passwords protect nothing, and a suite creates hundreds of users
	hash, err := bcrypt.GenerateFromPassword([]byte(attrs.password), bcrypt.MinCost)
	if err != nil {
		f.t.Fatalf("factory.User: hash password: %v", err)
	}

	user, err := f.queries.FactoryCreateUser(ctx, db.FactoryCreateUserParams{
		Email:    attrs.email,
		Name:     attrs.name,
		Password: string(hash),
		Roles:    attrs.roles,
		Verified: attrs.verified,
	})
	if err != nil {
		f.t.Fatalf("factory.User: %v", err)
	}
	return &user
}
