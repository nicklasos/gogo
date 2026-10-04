package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	AppName    string
	AppVersion string

	// Environment variables
	DatabaseURL     string
	TestDatabaseURL string
	RedisURL        string
	Port            string
	Environment     string
	Debug           bool
	LogLevel        string
	LogFormat       string
	LogOutput       string
	JWTSecret       string
	AppURL          string
	FilesBaseURL    string
	UploadFolder    string

	// CORSAllowedOrigins is empty (or contains "*") to allow any origin
	CORSAllowedOrigins []string

	// Scheduler configuration
	EnableScheduler bool
}

// Load loads configuration from environment
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if missing)
	_ = godotenv.Load()

	return &Config{
		AppName:    getEnv("APP_NAME", "MyApp"),
		AppVersion: "1.0.0",

		// Environment variables with defaults
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		TestDatabaseURL: getEnv("TEST_DATABASE_URL", ""),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/1"),
		Port:            getEnv("PORT", "8181"),
		Environment:     getEnv("APP_ENV", "development"),
		Debug:           getEnvBool("APP_DEBUG", false),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),
		LogOutput:       getEnv("LOG_OUTPUT", "both"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		AppURL:          getEnv("APP_URL", "localhost:8181"),
		FilesBaseURL:    getEnv("FILES_BASE_URL", fmt.Sprintf("http://localhost:%s/api/files", getEnv("PORT", "8181"))),
		UploadFolder:    getEnv("UPLOAD_FOLDER", "./uploads"),

		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS"),

		// Scheduler configuration
		EnableScheduler: getEnvBool("ENABLE_SCHEDULER", true),
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

func getEnvList(key string) []string {
	var values []string
	for _, item := range strings.Split(os.Getenv(key), ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
