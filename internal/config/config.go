package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration settings for the application.
type Config struct {
	Port     string
	DBPath   string
	LogLevel string
	Env      string
}

// Load reads configuration from environment variables and an optional .env file,
// returning a Config populated with sensible defaults for missing values.
func Load() (*Config, error) {
	// Attempt loading from .env; ignore error if file does not exist.
	_ = godotenv.Load()

	cfg := &Config{
		Port:     getEnv("PORT", "8080"),
		DBPath:   getEnv("DB_PATH", "data/walkthecity.db"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Env:      getEnv("ENV", "development"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
