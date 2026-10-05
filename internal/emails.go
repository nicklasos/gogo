package internal

import "strings"

// NormalizeEmail is the form an email is stored and looked up in: trimmed and lower-case.
// Apply it wherever an email enters the system, so "User@Example.com" and
// "user@example.com" are one account.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
