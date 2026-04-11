-- db/queries/user.sql

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: FindUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name, is_sso_user)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUserEmailVerified :one
UPDATE users
SET email_verified_at = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
