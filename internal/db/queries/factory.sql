-- These queries exist for internal/factory, which writes test and seed data. They may set
-- columns the API sets itself, so a factory can build any state a test needs. Application
-- code uses the other files.

-- name: FactoryCreateUser :one
INSERT INTO users (email, name, password, roles, email_verified_at)
VALUES (
    sqlc.arg(email), sqlc.arg(name), sqlc.arg(password), sqlc.arg(roles),
    CASE WHEN sqlc.arg(verified)::bool THEN CURRENT_TIMESTAMP END
)
RETURNING *;

-- name: FactoryCreateExample :one
INSERT INTO examples (user_id, title, description)
VALUES (sqlc.arg(user_id), sqlc.arg(title), sqlc.arg(description))
RETURNING *;

-- name: FactoryCreateUpload :one
INSERT INTO uploads (user_id, folder_id, type, relative_path, original_filename, file_size, mime_type)
VALUES (sqlc.arg(user_id), sqlc.arg(user_id), 'image', sqlc.arg(relative_path), 'test.jpg', 1024, 'image/jpeg')
RETURNING *;
