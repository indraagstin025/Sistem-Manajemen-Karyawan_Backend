package paseto

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"Sistem-Manajemen-Karyawan/config"
	"Sistem-Manajemen-Karyawan/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Claims struct {
	UserID       primitive.ObjectID `json:"user_id"`
	Email        string             `json:"email"`
	Role         string             `json:"role"`
	IsFirstLogin bool               `json:"is_first_login"`
}

var (
	pasetoInstance = paseto.NewV2()
	symmetricKey   []byte
)

func init() {
    cfg := config.LoadConfig()

    // Try decode with different base64 variants
    var decodedKey []byte
    var err error

    // Try standard base64 URL encoding first
    decodedKey, err = base64.URLEncoding.DecodeString(cfg.PASETO_SECRET)
    if err != nil {
        // Try with padding
        decodedKey, err = base64.URLEncoding.WithPadding(base64.StdPadding).DecodeString(cfg.PASETO_SECRET)
        if err != nil {
            // Try standard base64
            decodedKey, err = base64.StdEncoding.DecodeString(cfg.PASETO_SECRET)
            if err != nil {
                panic(fmt.Sprintf("Failed to decode PASETO_SECRET: %v", err))
            }
        }
    }

    if len(decodedKey) != 32 {
        panic(fmt.Sprintf("PASETO_SECRET must be exactly 32 bytes after Base64 decoding, got %d bytes", len(decodedKey)))
    }

    symmetricKey = decodedKey
}

func GenerateToken(user *models.User) (string, error) {
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	token := paseto.JSONToken{
		IssuedAt:   now,
		Expiration: exp,
		NotBefore:  now,
	}

	// Custom claims disimpan sebagai string
	token.Set("user_id", user.ID.Hex())
	token.Set("email", user.Email)
	token.Set("role", user.Role)
	token.Set("is_first_login", fmt.Sprintf("%v", user.IsFirstLogin)) // convert bool to string

	return pasetoInstance.Encrypt(symmetricKey, token, "")
}

func ValidateToken(tokenString string) (*Claims, error) {
	var token paseto.JSONToken
	var footer string

	err := pasetoInstance.Decrypt(tokenString, symmetricKey, &token, &footer)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt paseto token: %w", err)
	}

	if err := token.Validate(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	
	var claims Claims

	userIDStr := token.Get("user_id")
	objectID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %v", err)
	}
	claims.UserID = objectID
	claims.Email = token.Get("email")
	claims.Role = token.Get("role")
	claims.IsFirstLogin = (token.Get("is_first_login") == "true")

	return &claims, nil
}
