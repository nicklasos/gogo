-- name: GetCityByID :one
SELECT * FROM cities 
WHERE id = $1 LIMIT 1;

-- name: GetCityByName :one
SELECT * FROM cities 
WHERE name = $1 LIMIT 1;

-- name: ListCities :many
SELECT * FROM cities 
ORDER BY name ASC;

-- name: UpdateCityLatLon :exec
UPDATE cities 
SET lat = $2, lon = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;