-- db/queries/location.sql

-- name: FindLocationByID :one
SELECT * FROM locations WHERE id = $1 LIMIT 1;
