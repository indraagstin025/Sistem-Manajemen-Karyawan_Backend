package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QRCode struct {
	ID        primitive.ObjectID   `json:"id,omitempty" bson:"_id,omitempty"`
	Code      string               `json:"code" bson:"code,omitempty"`
	Date      string               `json:"date" bson:"date,omitempty"` 
	ExpiresAt time.Time            `json:"expires_at" bson:"expires_at,omitempty"`
	CreatedAt time.Time            `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time            `json:"updated_at" bson:"updated_at,omitempty"` 
}

type QRCodeGeneratePayload struct {
	Date string `json:"date" validate:"required,datetime=2006-01-02"`
}

type QRCodeScanPayload struct {
	QRCodeValue string `json:"qr_code_value" validate:"required"`
	UserID      string `json:"user_id" validate:"required"`
}
