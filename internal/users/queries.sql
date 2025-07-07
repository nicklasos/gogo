-- name: GetUserByID :one
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE id = $1;

-- name: CreateUser :execresult
INSERT INTO users (name, email) VALUES ($1, $2);

-- name: ListUsers :many
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users ORDER BY id;

-- name: GetUserByEmail :one
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE email = $1;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;

-- name: VerifyUserEmail :exec
UPDATE users SET email_verified = TRUE, email_verified_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: GetActiveUsers :many
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE status = 'active' ORDER BY id;