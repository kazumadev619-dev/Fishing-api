package entity

import (
	"time"

	"github.com/google/uuid"
)

type LocationType string

const (
	LocationTypeShore    LocationType = "SHORE"
	LocationTypeSurf     LocationType = "SURF"
	LocationTypePort     LocationType = "PORT"
	LocationTypeRiver    LocationType = "RIVER"
	LocationTypeLake     LocationType = "LAKE"
	LocationTypeOffshore LocationType = "OFFSHORE"
	LocationTypeOther    LocationType = "OTHER"
)

type Location struct {
	ID           uuid.UUID
	Name         string
	Latitude     float64
	Longitude    float64
	Region       *string
	Prefecture   *string
	LocationType LocationType
	PortID       *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Port struct {
	ID             uuid.UUID
	Name           string
	PrefectureCode string
	PrefectureName *string
	PortCode       string
	Latitude       *float64
	Longitude      *float64
	CreatedAt      time.Time
}
