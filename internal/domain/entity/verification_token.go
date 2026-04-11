package entity

import (
	"time"

	"github.com/google/uuid"
)

type VerificationToken struct {
	ID        uuid.UUID
	Email     string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
