package entity

import "time"

type TideEvent struct {
	Time   time.Time
	Height float64
}

type TideData struct {
	PortCode  string
	Date      string
	HighTides []TideEvent
	LowTides  []TideEvent
	TideType  string
}
