package users

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// UserService contains business logic and uses sqlc directly
type UserService struct {
	queries *Queries  // sqlc generated
	db      *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		queries: New(db),  // sqlc generated New function
		db:      db,
	}
}

// CreateUser handles user creation with business logic
func (us *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	// Business logic: validate input
	if err := us.validateUserInput(name, email); err != nil {
		return nil, err
	}
	
	// Business logic: check if user already exists
	exists, err := us.checkUserExists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}
	
	// Create user using sqlc directly
	params := CreateUserParams{
		Name:  name,
		Email: email,
	}
	
	result, err := us.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Get the created user
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get created user ID: %w", err)
	}
	
	user, err := us.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created user: %w", err)
	}
	
	// Business logic: perform post-creation tasks
	go us.sendWelcomeNotification(user.Email)
	
	return &user, nil
}

// GetUser retrieves user by ID with validation
func (us *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", id)
	}
	
	user, err := us.queries.GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return &user, nil
}

// GetAllUsers retrieves all users
func (us *UserService) GetAllUsers(ctx context.Context) ([]User, error) {
	return us.queries.ListUsers(ctx)
}

// UpdateUser handles user updates with business logic
func (us *UserService) UpdateUser(ctx context.Context, id int64, name, email string) (*User, error) {
	// Business logic: validate input
	if err := us.validateUserInput(name, email); err != nil {
		return nil, err
	}
	
	// Business logic: check if user exists
	exists, err := us.userExists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("user with ID %d not found", id)
	}
	
	// TODO: Add UpdateUser SQL query to queries.sql
	// For now, this is a placeholder
	return nil, fmt.Errorf("update user not implemented yet")
}

// Business logic helper methods

func (us *UserService) validateUserInput(name, email string) error {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) < 2 {
		return fmt.Errorf("name must be at least 2 characters long")
	}
	if len(name) > 100 {
		return fmt.Errorf("name cannot exceed 100 characters")
	}
	
	// Validate email
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !us.isValidEmail(email) {
		return fmt.Errorf("invalid email format")
	}
	
	return nil
}

func (us *UserService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func (us *UserService) checkUserExists(ctx context.Context, email string) (bool, error) {
	// TODO: Add GetUserByEmail SQL query to queries.sql
	// For now, return false (user doesn't exist)
	return false, nil
}

func (us *UserService) userExists(ctx context.Context, id int64) (bool, error) {
	_, err := us.queries.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Async business logic - would typically use a queue in production
func (us *UserService) sendWelcomeNotification(email string) {
	// TODO: Implement welcome email/notification
	fmt.Printf("Sending welcome notification to: %s\n", email)
}