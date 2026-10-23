-- name: Healthcheck :exec
SELECT 1;

-- name: DatabaseVersion :one
SELECT version()::text AS version;
