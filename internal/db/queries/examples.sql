-- name: GetExampleByID :one
SELECT * FROM examples
WHERE id = $1 LIMIT 1;

-- name: CreateExample :one
INSERT INTO examples (
    user_id, title, description
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: UpdateExample :one
UPDATE examples
SET
    title = $2,
    description = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteExample :exec
DELETE FROM examples
WHERE id = $1;

-- name: ListExamplesForUserPaginated :many
SELECT * FROM examples
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountExamplesForUser :one
SELECT COUNT(*) FROM examples
WHERE user_id = $1;
