package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Stripe configs
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string

	// Server configs
	Port        string
	Environment string

	// Additional configs
	CorsAllowedOrigins []string
	LogLevel           string
}

// Load initializes configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists
	godotenv.Load()

	// Initialize with default values
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	// Required Stripe keys
	config.StripeSecretKey = mustGetEnv("STRIPE_SECRET_KEY")
	config.StripePublishableKey = getEnv("STRIPE_PUBLISHABLE_KEY", "")
	config.StripeWebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", "")

	// Parse CORS allowed origins
	corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", "")
	if corsOrigins != "" {
		config.CorsAllowedOrigins = strings.Split(corsOrigins, ",")
	} else {
		config.CorsAllowedOrigins = []string{"*"}
	}

	return config
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// mustGetEnv gets an environment variable or panics if it's not set
func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable not set: %s", key)
	}
	return value
}