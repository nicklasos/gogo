-- name: GetCityByID :one
SELECT id, name, created_at, updated_at FROM cities 
WHERE id = $1 LIMIT 1;

-- name: GetCityByName :one
SELECT id, name, created_at, updated_at FROM cities 
WHERE name = $1 LIMIT 1;

-- name: ListCities :many
SELECT id, name, created_at, updated_at FROM cities 
ORDER BY name ASC;