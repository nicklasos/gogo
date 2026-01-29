-- name: CreateUpload :one
INSERT INTO uploads (
    user_id, folder_id, type, relative_path, original_filename, file_size, mime_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetUploadByID :one
SELECT * FROM uploads
WHERE id = $1 LIMIT 1;

-- name: GetUploadByIDAndUserID :one
SELECT * FROM uploads
WHERE id = $1 AND user_id = $2 LIMIT 1;

-- name: ListUploadsByUserID :many
SELECT * FROM uploads
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListUploadsByFolderID :many
SELECT * FROM uploads
WHERE folder_id = $1
ORDER BY created_at DESC;

-- name: DeleteUpload :exec
DELETE FROM uploads
WHERE id = $1 AND user_id = $2;

-- name: GetUploadByPath :one
SELECT * FROM uploads
WHERE relative_path = $1 LIMIT 1;
