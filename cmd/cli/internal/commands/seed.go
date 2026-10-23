package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"app/cmd/cli/internal"
	"app/internal/factory"
	"app/internal/middleware"

	"github.com/jackc/pgx/v5"
)

const seedAdminEmail = "admin@example.com"

// seedReporter lets the factories, which report failures the way tests do, run from the CLI.
type seedReporter struct{}

func (seedReporter) Helper() {}
func (seedReporter) Fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Seeding failed: "+format+"\n", args...)
	os.Exit(1)
}

// RunSeed fills an empty development database with accounts and sample data.
// Add the records of a new module here, built with its factory.
func RunSeed(app *internal.CLIApp, args []string) {
	if app.Config.IsProduction() {
		fmt.Fprintln(os.Stderr, "Refusing to seed: APP_ENV is production")
		os.Exit(1)
	}

	ctx := context.Background()
	if _, err := app.Queries.GetUserByEmail(ctx, seedAdminEmail); err == nil {
		fmt.Printf("Already seeded: %s exists. Nothing was changed.\n", seedAdminEmail)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "Seeding failed: %v\n", err)
		os.Exit(1)
	}

	create := factory.New(seedReporter{}, app.Database)

	create.User(factory.WithEmail(seedAdminEmail), factory.WithName("Super Admin"), factory.WithRoles(middleware.RoleSuperAdmin))
	create.User(factory.WithEmail("manager@example.com"), factory.WithName("Manager"), factory.WithRoles(middleware.RoleAdmin))

	user := create.User(factory.WithEmail("user@example.com"), factory.WithName("Demo User"))
	create.User(factory.WithEmail("unverified@example.com"), factory.WithName("New Signup"), factory.Unverified())

	create.Example(user.ID,
		factory.WithTitle("Welcome"),
		factory.WithDescription("## Hello\n\nThis example was created by `make seed`. Its description is **markdown**.\n\n- Edit it\n- Delete it\n- Add your own"),
	)
	create.Examples(user.ID, 30, "Sample example")

	fmt.Println("Seeded. Every account has the password:", factory.DefaultPassword)
	fmt.Println("  super admin  ", seedAdminEmail)
	fmt.Println("  admin         manager@example.com")
	fmt.Println("  user          user@example.com (owns 31 examples)")
	fmt.Println("  user          unverified@example.com (email not confirmed)")
}
