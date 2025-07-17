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
	RequestType   string             `json:"request_type" bson:"request_type"` // "Cuti", "Sakit", "Izin"
	AttachmentURL string             `json:"attachment_url,omitempty" bson:"attachment_url,omitempty"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

// BARU: Tambahkan struct ini ke file models/leave_request.go Anda
type LeaveRequestWithUser struct {
	LeaveRequest `bson:",inline"` 
	UserName     string           `json:"user_name" bson:"user_name"`    // <--- UBAH DARI "user_info.name" MENJADI "user_name"
	UserEmail    string           `json:"user_email" bson:"user_email"`  // <--- UBAH DARI "user_info.email" MENJADI "user_email"
	UserPhoto    string           `json:"user_photo,omitempty" bson:"user_photo,omitempty"` // <--- UBAH DARI "user_info.photo" MENJADI "user_photo"
}

type LeaveSummaryResponse struct {
	CurrentMonthLeaveCount int64 `json:"current_month_leave_count"`
	AnnualLeaveCount       int64 `json:"annual_leave_count"`
}

type LeaveRequestCreatePayload struct {
	UserID      string `json:"user_id" validate:"required"`
	StartDate   string `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate     string `json:"end_date" validate:"required,datetime=2006-01-02,gtefield=StartDate"`
	RequestType string `json:"request_type" validate:"required,oneof=Cuti Sakit"`
	Reason      string `json:"reason" validate:"required,min=10,max=500"`
}



type LeaveRequestUpdatePayload struct {
	Status string `json:"status" validate:"required,oneof=pending approved rejected"`
	Note   string `json:"note,omitempty"`
}