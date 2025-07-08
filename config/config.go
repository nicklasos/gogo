package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Hardcoded constants
	AppName    string
	AppVersion string

	// Environment variables
	DatabaseURL string
	RedisURL    string
	Port        string
	Environment string
	Debug       bool
	LogLevel    string
	LogFormat   string
	LogOutput   string
	JWTSecret   string
}

// Load loads configuration from environment
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if missing)
	_ = godotenv.Load()

	return &Config{
		// Hardcoded values
		AppName:    "MyApp",
		AppVersion: "1.0.0",

		// Environment variables with defaults
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("APP_ENV", "development"),
		Debug:       getEnvBool("APP_DEBUG", false),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogFormat:   getEnv("LOG_FORMAT", "json"),
		LogOutput:   getEnv("LOG_OUTPUT", "both"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
	}, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}