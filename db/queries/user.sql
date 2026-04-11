-- db/queries/user.sql

-- name: FindUserByEmail :one
SELECT id, email, password_hash, name, avatar_url, is_sso_user, email_verified_at, created_at, updated_at
FROM users WHERE email = $1 LIMIT 1;

-- name: FindUserByID :one
SELECT id, email, password_hash, name, avatar_url, is_sso_user, email_verified_at, created_at, updated_at
FROM users WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name, is_sso_user)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, password_hash, name, avatar_url, is_sso_user, email_verified_at, created_at, updated_at;

-- name: UpdateUserEmailVerified :one
UPDATE users
SET email_verified_at = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, password_hash, name, avatar_url, is_sso_user, email_verified_at, created_at, updated_at;
