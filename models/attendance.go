package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Attendance struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id,omitempty"`
	Date      string             `json:"date" bson:"date,omitempty"`
	CheckIn   string             `json:"check_in" bson:"check_in,omitempty"`
	CheckOut  string             `json:"check_out" bson:"check_out,omitempty"`
	Status    string             `json:"status" bson:"status,omitempty"`
	Note      string             `json:"note" bson:"note,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

type AttendanceCreatePayload struct {
	UserID   string `json:"user_id" validate:"required"`
	Date     string `json:"date" validate:"required,datetime=2006-01-02"`
	CheckIn  string `json:"check_in" validate:"required,datetime=15:04"`
	CheckOut string `json:"check_out" validate:"omitempty,datetime=15:04"`
	Status   string `json:"status" validate:"required,oneof=Hadir Telat Izin Sakit Cuti Alpha"`
	Note     string `json:"note"`
}

type AttendanceUpdatePayload struct {
	CheckIn  string `json:"check_in,omitempty" validate:"omitempty,datetime=15:04"`
	CheckOut string `json:"check_out,omitempty" validate:"omitempty,datetime=15:04"`
	Status   string `json:"status,omitempty" validate:"omitempty,oneof=Hadir Telat Izin Sakit Cuti Alpha"`
	Note     string `json:"note,omitempty"`
}
