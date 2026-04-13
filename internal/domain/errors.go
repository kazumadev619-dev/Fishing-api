package domain

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrEmailNotVerified = errors.New("email not verified")
)
