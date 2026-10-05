-- name: GetUserByID :one
SELECT * FROM users 
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE email = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    email, name, password, roles, email_verified_at
) VALUES (
    sqlc.arg(email), sqlc.arg(name), sqlc.arg(password), sqlc.arg(roles),
    CASE WHEN sqlc.arg(email_verified)::bool THEN CURRENT_TIMESTAMP END
)
RETURNING *;

-- Newest first, so an account that was just created is on the first page.
-- name: ListUsersByRole :many
SELECT * FROM users
WHERE sqlc.arg(role)::text = ANY(roles)
ORDER BY id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsersByRole :one
SELECT count(*) FROM users
WHERE sqlc.arg(role)::text = ANY(roles);

-- A changed email is unverified again unless keep_verified is set (an admin vouches for it).
-- name: UpdateUserProfile :one
UPDATE users
SET
    email_verified_at = CASE
        WHEN email = sqlc.arg(email) OR sqlc.arg(keep_verified)::bool THEN email_verified_at
    END,
    email = sqlc.arg(email),
    name = sqlc.arg(name),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: MarkUserEmailVerified :exec
UPDATE users
SET email_verified_at = COALESCE(email_verified_at, CURRENT_TIMESTAMP)
WHERE id = $1;

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

-- Auth Token Queries (password reset, email verification)
-- name: CreateAuthToken :exec
INSERT INTO auth_tokens (
    user_id, purpose, token_hash, expires_at
) VALUES (
    sqlc.arg(user_id), sqlc.arg(purpose), sqlc.arg(token_hash),
    CURRENT_TIMESTAMP + sqlc.arg(ttl_seconds)::int * INTERVAL '1 second'
);

-- Marks the token used and returns its owner in one statement, so a link works exactly once.
-- name: ConsumeAuthToken :one
UPDATE auth_tokens
SET used_at = CURRENT_TIMESTAMP
WHERE token_hash = sqlc.arg(token_hash)
  AND purpose = sqlc.arg(purpose)
  AND used_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP
RETURNING user_id;

-- name: DeleteUserAuthTokens :exec
DELETE FROM auth_tokens
WHERE user_id = sqlc.arg(user_id) AND purpose = sqlc.arg(purpose);

-- name: DeleteExpiredAuthTokens :exec
DELETE FROM auth_tokens
WHERE expires_at < CURRENT_TIMESTAMP OR used_at IS NOT NULL;
