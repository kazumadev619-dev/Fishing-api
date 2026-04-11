-- db/queries/verification_token.sql

-- name: CreateVerificationToken :one
INSERT INTO verification_tokens (id, email, token, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, email, token, expires_at, created_at;

-- name: FindVerificationToken :one
SELECT id, email, token, expires_at, created_at
FROM verification_tokens WHERE token = $1 LIMIT 1;

-- name: DeleteVerificationTokensByEmail :exec
DELETE FROM verification_tokens WHERE email = $1;

-- name: DeleteExpiredVerificationTokens :exec
DELETE FROM verification_tokens WHERE expires_at < NOW();
