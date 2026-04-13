package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	External ExternalConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port       string
	AppBaseURL string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
}

type ExternalConfig struct {
	OpenWeatherAPIKey string
	GoogleMapsAPIKey  string
}

type EmailConfig struct {
	ResendAPIKey string
	FromAddress  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:       getEnv("PORT", "8080"),
			AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			URL: os.Getenv("DATABASE_URL"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		JWT: JWTConfig{
			AccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
			RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		},
		External: ExternalConfig{
			OpenWeatherAPIKey: os.Getenv("OPENWEATHER_API_KEY"),
			GoogleMapsAPIKey:  os.Getenv("GOOGLE_MAPS_API_KEY"),
		},
		Email: EmailConfig{
			ResendAPIKey: os.Getenv("RESEND_API_KEY"),
			FromAddress:  getEnv("EMAIL_FROM", "noreply@fishing-app.com"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	required := []struct {
		key string
		val string
	}{
		{"DATABASE_URL", c.Database.URL},
		{"JWT_ACCESS_SECRET", c.JWT.AccessSecret},
		{"JWT_REFRESH_SECRET", c.JWT.RefreshSecret},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("required environment variable not set: %s", r.key)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
