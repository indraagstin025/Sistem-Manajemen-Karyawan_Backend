package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LeaveRequest struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id,omitempty"`
	StartDate     string             `json:"start_date" bson:"start_date,omitempty"`
	EndDate       string             `json:"end_date" bson:"end_date,omitempty"`
	Reason        string             `json:"reason" bson:"reason,omitempty"`
	Status        string             `json:"status" bson:"status,omitempty"`
	Note          string             `json:"note" bson:"note,omitempty"`
	RequestType  string             `json:"request_type" bson:"request_type"` // TAMBAHAN: "Cuti", "Sakit", "Izin"
	AttachmentURL string             `json:"attachment_url,omitempty" bson:"attachment_url,omitempty"`
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

type LeaveRequestCreatePayload struct {
	UserID    string `json:"user_id" validate:"required"`
	StartDate string `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate   string `json:"end_date" validate:"required,datetime=2006-01-02,gtefield=StartDate"`
	RequestType string `json:"request_type" validate:"required,oneof=Cuti Sakit Izin"` // TAMBAHAN
	Reason    string `json:"reason" validate:"required,min=10,max=500"`
}

type LeaveRequestUpdatePayload struct {
	Status string `json:"status" validate:"required,oneof=pending approved rejected"`
	Note   string `json:"note,omitempty"`
}
