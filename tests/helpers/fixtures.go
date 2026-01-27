package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"app/internal/db"
	"app/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// GetTestTimestamp returns a test timestamp
func GetTestTimestamp() pgtype.Timestamp {
	return pgtype.Timestamp{Time: time.Now(), Valid: true}
}

// CreateTestUser creates a test user and returns it
func CreateTestUser(t *testing.T, ctx context.Context, tx pgx.Tx) *db.User {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Generate unique email using timestamp
	email := fmt.Sprintf("test%d@example.com", time.Now().UnixNano())

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	user := &db.User{
		Email:     email,
		Name:      "Test User",
		Password:  string(hashedPassword),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Insert test user
	row := tx.QueryRow(ctx,
		"INSERT INTO users (email, name, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, email, name, password, roles, created_at, updated_at",
		user.Email, user.Name, user.Password, user.CreatedAt, user.UpdatedAt)

	err = row.Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.Roles, &user.CreatedAt, &user.UpdatedAt)
	require.NoError(t, err, "Failed to create test user")

	return user
}

// CreateTestUserWithEmail creates a test user with a specific email
func CreateTestUserWithEmail(t *testing.T, ctx context.Context, tx pgx.Tx, email string) *db.User {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	user := &db.User{
		Email:     email,
		Name:      "Test User",
		Password:  string(hashedPassword),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Insert test user
	row := tx.QueryRow(ctx,
		"INSERT INTO users (email, name, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, email, name, password, roles, created_at, updated_at",
		user.Email, user.Name, user.Password, user.CreatedAt, user.UpdatedAt)

	err = row.Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.Roles, &user.CreatedAt, &user.UpdatedAt)
	require.NoError(t, err, "Failed to create test user")

	return user
}

// CreateTestExample creates a test example and returns it
func CreateTestExample(t *testing.T, ctx context.Context, tx pgx.Tx, userID int32) *db.Example {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Generate unique title using timestamp
	title := fmt.Sprintf("Test Example %d", time.Now().UnixNano())

	example := &db.Example{
		UserID:      userID,
		Title:        title,
		Description:  pgtype.Text{String: "Test description", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Insert test example
	row := tx.QueryRow(ctx,
		"INSERT INTO examples (user_id, title, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, title, description, created_at, updated_at",
		example.UserID, example.Title, example.Description, example.CreatedAt, example.UpdatedAt)

	err := row.Scan(&example.ID, &example.UserID, &example.Title, &example.Description, &example.CreatedAt, &example.UpdatedAt)
	require.NoError(t, err, "Failed to create test example")

	return example
}

// CreateTestExampleWithTitle creates a test example with a specific title
func CreateTestExampleWithTitle(t *testing.T, ctx context.Context, tx pgx.Tx, userID int32, title string) *db.Example {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	example := &db.Example{
		UserID:      userID,
		Title:        title,
		Description:  pgtype.Text{String: "Test description", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Insert test example
	row := tx.QueryRow(ctx,
		"INSERT INTO examples (user_id, title, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, title, description, created_at, updated_at",
		example.UserID, example.Title, example.Description, example.CreatedAt, example.UpdatedAt)

	err := row.Scan(&example.ID, &example.UserID, &example.Title, &example.Description, &example.CreatedAt, &example.UpdatedAt)
	require.NoError(t, err, "Failed to create test example")

	return example
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(t *testing.T, ctx context.Context, tx pgx.Tx, email string) *db.User {
	var user db.User
	row := tx.QueryRow(ctx,
		"SELECT id, email, name, password, roles, created_at, updated_at FROM users WHERE email = $1",
		email)
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.Roles, &user.CreatedAt, &user.UpdatedAt)
	require.NoError(t, err, "Failed to get user by email")
	return &user
}

// GetTestLogger creates a test logger
func GetTestLogger(t *testing.T) *logger.Logger {
	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create test logger")
	return testLogger
}
