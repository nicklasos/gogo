package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"app/cmd/cli/internal"
	appinternal "app/internal"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

// RunCreateUser creates a user in the database.
func RunCreateUser(app *internal.CLIApp, args []string) {
	fs := flag.NewFlagSet("create-user", flag.ExitOnError)
	email := fs.String("email", "", "User email (required)")
	name := fs.String("name", "", "User name (defaults to the email local part)")
	password := fs.String("password", "", "User password, min 8 characters (required)")
	role := fs.String("role", middleware.RoleSuperAdmin, "User role: "+strings.Join(middleware.Roles, ", "))
	fs.Usage = func() {
		fmt.Println("Usage: go run ./cmd/cli create-user --email EMAIL --password PASSWORD [--name NAME] [--role ROLE]")
		fmt.Println()
		fmt.Println("Create a user. Default role is super-admin.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run ./cmd/cli create-user --email admin@example.com --password password123")
		fmt.Println("  go run ./cmd/cli create-user --email admin@example.com --name Admin --password password123 --role admin")
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
		os.Exit(1)
	}

	*email = appinternal.NormalizeEmail(*email)
	*name = strings.TrimSpace(*name)
	*role = strings.TrimSpace(*role)

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Error: --email and --password are required")
		fs.Usage()
		os.Exit(1)
	}
	if len(*password) < 8 {
		fmt.Fprintln(os.Stderr, "Error: --password must be at least 8 characters")
		os.Exit(1)
	}
	if !slices.Contains(middleware.Roles, *role) {
		fmt.Fprintf(os.Stderr, "Error: unknown role %q (expected one of: %s)\n", *role, strings.Join(middleware.Roles, ", "))
		os.Exit(1)
	}
	if *name == "" {
		*name = strings.SplitN(*email, "@", 2)[0]
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	user, err := app.Queries.CreateUser(context.Background(), db.CreateUserParams{
		Email:    *email,
		Name:     *name,
		Password: string(hashedPassword),
		Roles:    []string{*role},

		EmailVerified: true,
	})
	if err != nil {
		if domainErr := errs.DomainErrorFromPostgresUniqueViolation(err); domainErr != nil {
			fmt.Fprintf(os.Stderr, "User with this email already exists: %s\n", *email)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Failed to create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created user id=%d email=%s name=%s roles=%s\n", user.ID, user.Email, user.Name, strings.Join(user.Roles, ","))
}
