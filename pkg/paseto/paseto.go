package paseto

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"Sistem-Manajemen-Karyawan/models"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

func NewPasetoMaker() (*PasetoMaker, error) {
	secretBase64 := os.Getenv("PASETO_SECRET")
	if secretBase64 == "" {
		return nil, fmt.Errorf("PASETO_SECRET tidak ditemukan di environment variables")
	}

	// SESUDAH
	decodedKey, err := base64.StdEncoding.DecodeString(secretBase64)
	if err != nil {
		return nil, fmt.Errorf("gagal decode PASETO_SECRET dari base64: %w", err)
	}

	if len(decodedKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("kunci rahasia harus tepat %d bytes setelah di-decode, bukan %d", chacha20poly1305.KeySize, len(decodedKey))
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: decodedKey,
	}

	return maker, nil
}

func (maker *PasetoMaker) GenerateToken(user *models.User) (string, error) {
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	token := paseto.JSONToken{
		IssuedAt:   now,
		Expiration: exp,
		NotBefore:  now,
	}

	token.Set("user_id", user.ID.Hex())
	token.Set("email", user.Email)
	token.Set("role", user.Role)
	token.Set("is_first_login", fmt.Sprintf("%v", user.IsFirstLogin))

	return maker.paseto.Encrypt(maker.symmetricKey, token, nil)
}

func (maker *PasetoMaker) ValidateToken(tokenString string) (*models.Claims, error) {
	var token paseto.JSONToken

	err := maker.paseto.Decrypt(tokenString, maker.symmetricKey, &token, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal decrypt token: %w", err)
	}

	err = token.Validate()
	if err != nil {
		return nil, fmt.Errorf("validasi token gagal: %w", err)
	}

	claims := &models.Claims{}
	userIDHex := token.Get("user_id")
	objectID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, fmt.Errorf("format user_id tidak valid: %w", err)
	}

	claims.UserID = objectID
	claims.Email = token.Get("email")
	claims.Role = token.Get("role")
	claims.IsFirstLogin = (token.Get("is_first_login") == "true")

	return claims, nil
}
