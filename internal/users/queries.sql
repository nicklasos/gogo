-- name: GetUserByID :one
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE id = ?;

-- name: CreateUser :execresult
INSERT INTO users (name, email) VALUES (?, ?);

-- name: ListUsers :many
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users ORDER BY id;

-- name: GetUserByEmail :one
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE email = ?;

-- name: UpdateUserStatus :exec
UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: VerifyUserEmail :exec
UPDATE users SET email_verified = TRUE, email_verified_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: GetActiveUsers :many
SELECT id, name, email, status, email_verified, email_verified_at, created_at, updated_at 
FROM users WHERE status = 'active' ORDER BY id;