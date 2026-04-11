-- db/queries/location.sql

-- name: FindLocationByID :one
SELECT id, name, latitude, longitude, region, prefecture, location_type, port_id, created_at, updated_at
FROM locations WHERE id = $1 LIMIT 1;
