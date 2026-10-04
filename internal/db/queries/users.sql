-- name: GetUserByID :one
SELECT * FROM users 
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE email = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    email, name, password, roles
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: ListUsersByRole :many
SELECT * FROM users
WHERE sqlc.arg(role)::text = ANY(roles)
ORDER BY id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsersByRole :one
SELECT count(*) FROM users
WHERE sqlc.arg(role)::text = ANY(roles);

-- name: UpdateUserProfile :one
UPDATE users
SET
    email = $2,
    name = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- Refresh Token Queries
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id, token, expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND expires_at > NOW() AND is_revoked = FALSE
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW();
