package factory

import (
	"fmt"

	"app/internal/db"
)

type exampleAttrs struct {
	title       string
	description string
}

type ExampleOption func(*exampleAttrs)

func WithTitle(title string) ExampleOption { return func(a *exampleAttrs) { a.title = title } }
func WithDescription(description string) ExampleOption {
	return func(a *exampleAttrs) { a.description = description }
}

// Example creates an example owned by the given user.
func Example(t TB, conn DB, userID int32, opts ...ExampleOption) *db.Example {
	t.Helper()

	attrs := exampleAttrs{
		title:       "Example " + unique(),
		description: "Test description",
	}
	for _, opt := range opts {
		opt(&attrs)
	}

	var example db.Example
	err := conn.QueryRow(ctx,
		`INSERT INTO examples (user_id, title, description) VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, description, created_at, updated_at`,
		userID, attrs.title, attrs.description,
	).Scan(&example.ID, &example.UserID, &example.Title, &example.Description, &example.CreatedAt, &example.UpdatedAt)
	if err != nil {
		t.Fatalf("factory.Example: %v", err)
	}
	return &example
}

// Examples creates count examples for the user, titled "<prefix> 001", "<prefix> 002", ...
func Examples(t TB, conn DB, userID int32, count int, prefix string) []*db.Example {
	t.Helper()

	examples := make([]*db.Example, count)
	for i := range examples {
		examples[i] = Example(t, conn, userID, WithTitle(fmt.Sprintf("%s %03d", prefix, i+1)))
	}
	return examples
}
