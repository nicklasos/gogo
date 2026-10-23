package factory

import (
	"fmt"

	"app/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
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
func (f *Factory) Example(userID int32, opts ...ExampleOption) *db.Example {
	f.t.Helper()

	attrs := exampleAttrs{
		title:       "Example " + unique(),
		description: "Test description",
	}
	for _, opt := range opts {
		opt(&attrs)
	}

	example, err := f.queries.FactoryCreateExample(ctx, db.FactoryCreateExampleParams{
		UserID:      userID,
		Title:       attrs.title,
		Description: pgtype.Text{String: attrs.description, Valid: true},
	})
	if err != nil {
		f.t.Fatalf("factory.Example: %v", err)
	}
	return &example
}

// Examples creates count examples for the user, titled "<prefix> 001", "<prefix> 002", ...
func (f *Factory) Examples(userID int32, count int, prefix string) []*db.Example {
	f.t.Helper()

	examples := make([]*db.Example, count)
	for i := range examples {
		examples[i] = f.Example(userID, WithTitle(fmt.Sprintf("%s %03d", prefix, i+1)))
	}
	return examples
}
