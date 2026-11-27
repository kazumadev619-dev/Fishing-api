package entity

import "time"

// WeatherData は現在の天気。Next の FormattedWeatherData と同じ中身。
// 単位は取得時の units（standard / metric / imperial）に従う。
type WeatherData struct {
	Lat, Lon    float64
	CityName    string
	Country     string
	Temperature float64
	FeelsLike   float64
	TempMin     float64
	TempMax     float64
	Humidity    int
	Pressure    float64
	WindSpeed   float64
	WindDeg     int
	WindGust    *float64 // 上流に無いときは nil
	Cloudiness  int
	Visibility  int
	Condition   WeatherCondition
	Rain        *float64 // 直近1時間（mm）。無いときは nil
	Snow        *float64
	Sunrise     time.Time
	Sunset      time.Time
	DateTime    time.Time
	Timezone    int // UTC からのずれ（秒）
}

type WeatherCondition struct {
	ID          int
	Main        string
	Description string
	Icon        string
}
