// Sistem-Manajemen-Karyawan/config/config.go
package config

import (
	"encoding/base64" // PERBAIKAN: Tambahkan import ini
	"log"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig holds the application-wide configurations
type AppConfig struct {
	Port          string // PERBAIKAN: Gunakan Port jika itu yang Anda inginkan
	MONGOSTRING   string
	PASETO_SECRET string
}

// LoadConfig loads configuration from .env file
func LoadConfig() *AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file (might not exist in production): %v", err)
	}

	secretBase64 := getEnv("PASETO_SECRET", "default_paseto_secret_base64_mustbe32bytes_") // PERBAIKAN: Default string base64 yang valid jika ingin pakai default
    // Pastikan default ini juga merupakan string base64 dari 32 byte jika digunakan sebagai fallback
    // Contoh: Base64 encode "01234567890123456789012345678901" (32 byte) menjadi "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE="
    // Lebih baik tidak pakai default jika ini untuk production.

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
		MONGOSTRING:   getEnv("MONGOSTRING", ""), // Ini adalah string URI MongoDB, jadi tidak ada default aman.
		PASETO_SECRET: secretBase64, // PERBAIKAN: Gunakan secretBase64 yang sudah dibaca
	}
}

// Helper function to get environment variable or fallback to default
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}