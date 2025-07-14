package models


import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkSchedule struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID     primitive.ObjectID `json:"user_id" bson:"user_id"`                      // relasi ke User
	Date       string             `json:"date" bson:"date"`                            // Format: "2006-01-02"
	StartTime  string             `json:"start_time" bson:"start_time"`                // Format: "15:04"
	EndTime    string             `json:"end_time" bson:"end_time"`                    // Format: "15:04"
	Note       string             `json:"note,omitempty" bson:"note,omitempty"`        // opsional: catatan (misal shift malam)
	CreatedAt  time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

type WorkScheduleCreatePayload struct {
	UserID    string `json:"user_id" validate:"required"`
	Date      string `json:"date" validate:"required,datetime=2006-01-02"`
	StartTime string `json:"start_time" validate:"required,datetime=15:04"`
	EndTime   string `json:"end_time" validate:"required,datetime=15:04"`
	Note      string `json:"note"`
}


type WorkScheduleUpdatePayload struct {
	Date      string `json:"date,omitempty" validate:"omitempty,datetime=2006-01-02"`
	StartTime string `json:"start_time,omitempty" validate:"omitempty,datetime=15:04"`
	EndTime   string `json:"end_time,omitempty" validate:"omitempty,datetime=15:04"`
	Note      string `json:"note,omitempty"`
}
