package factory

import (
	"app/internal/db"

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

// Unverified leaves the email unconfirmed, as it is right after self-registration.
func Unverified() UserOption { return func(a *userAttrs) { a.verified = false } }

// User creates a user with the role "user", a verified email and DefaultPassword.
func User(t TB, conn DB, opts ...UserOption) *db.User {
	t.Helper()

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
		t.Fatalf("factory.User: hash password: %v", err)
	}

	var user db.User
	err = conn.QueryRow(ctx,
		`INSERT INTO users (email, name, password, roles, email_verified_at)
		 VALUES ($1, $2, $3, $4, CASE WHEN $5 THEN CURRENT_TIMESTAMP END)
		 RETURNING id, email, name, password, roles, created_at, updated_at, email_verified_at`,
		attrs.email, attrs.name, string(hash), attrs.roles, attrs.verified,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.Roles, &user.CreatedAt, &user.UpdatedAt, &user.EmailVerifiedAt)
	if err != nil {
		t.Fatalf("factory.User: %v", err)
	}
	return &user
}
