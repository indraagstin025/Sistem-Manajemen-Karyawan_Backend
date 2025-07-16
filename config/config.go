
package config

import (
	"encoding/base64" 
	"log"
	"os"

	"github.com/joho/godotenv"
)


type AppConfig struct {
	Port          string 
	MONGOSTRING   string
	PASETO_SECRET string
}

// LoadConfig loads configuration from .env file
func LoadConfig() *AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file (might not exist in production): %v", err)
	}

	secretBase64 := getEnv("PASETO_SECRET", "default_paseto_secret_base64_mustbe32bytes_") 

	// Lakukan decoding untuk validasi panjang byte
	secretBytes, err := base64.URLEncoding.DecodeString(secretBase64)
	if err != nil {
		log.Fatalf("PASETO_SECRET in .env is not a valid Base64 URL-encoded string: %v", err)
	}

	if len(secretBytes) != 32 {
		log.Fatalf("PASETO_SECRET (decoded) must be exactly 32 bytes long. Current length: %d", len(secretBytes))
	}

	return &AppConfig{
		Port:          getEnv("PORT", "3000"),
		MONGOSTRING:   getEnv("MONGOSTRING", ""), 
		PASETO_SECRET: secretBase64, 
	}
}

// Helper function to get environment variable or fallback to default
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}