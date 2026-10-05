package users

import "app/internal"

// CreateUserRequest creates a user with a single role
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=super-admin admin user"`
}

// UpdateUserRequest updates name and email
type UpdateUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

// SetPasswordRequest sets a new password for a user
type SetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

// UserResponse is a user without the password
type UserResponse struct {
	ID            int32    `json:"id"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Roles         []string `json:"roles"`
	EmailVerified bool     `json:"email_verified"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// UserDataResponse wraps a user
type UserDataResponse struct {
	Data UserResponse `json:"data"`
}

// PaginatedUsersResponse wraps a page of users
type PaginatedUsersResponse struct {
	Data       []UserResponse          `json:"data"`
	Pagination internal.PaginationMeta `json:"pagination"`
}
