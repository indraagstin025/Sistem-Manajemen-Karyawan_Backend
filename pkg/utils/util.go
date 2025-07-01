package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateBase64Key generates a secure 32-byte key and returns it as base64 URL-encoded
func GenerateBase64Key(size int) (string, error) {
	if size != 32 {
		return "", fmt.Errorf("PASETO v2 local requires a 32-byte key")
	}

	key := make([]byte, size)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return base64.URLEncoding.EncodeToString(key), nil
}
