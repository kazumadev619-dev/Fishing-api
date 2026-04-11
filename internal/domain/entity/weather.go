package entity

import "time"

type WeatherData struct {
	Temperature float64
	FeelsLike   float64
	WindSpeed   float64
	WindDeg     int
	Pressure    float64
	Humidity    int
	Description string
	DateTime    time.Time
}
