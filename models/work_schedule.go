package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WorkSchedule sekarang menjadi jadwal umum, dengan dukungan aturan perulangan.
type WorkSchedule struct {
	ID             primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Date           string             `json:"date" bson:"date"`                 // Akan menjadi tanggal MULAI jadwal
	StartTime      string             `json:"start_time" bson:"start_time"`
	EndTime        string             `json:"end_time" bson:"end_time"`
	Note           string             `json:"note,omitempty" bson:"note,omitempty"`
	RecurrenceRule string             `json:"recurrence_rule,omitempty" bson:"recurrence_rule,omitempty"` // <-- TAMBAHKAN INI
	CreatedAt      time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

// WorkScheduleCreatePayload juga perlu menerima RecurrenceRule dari frontend.
type WorkScheduleCreatePayload struct {
	Date           string `json:"date" validate:"required,datetime=2006-01-02"`
	StartTime      string `json:"start_time" validate:"required,datetime=15:04"`
	EndTime        string `json:"end_time" validate:"required,datetime=15:04"`
	Note           string `json:"note"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"` // <-- TAMBAHKAN INI
}

// WorkScheduleUpdatePayload sekarang juga perlu menerima Date, StartTime, EndTime, Note, dan RecurrenceRule.
// Saya akan perbaiki ini agar sesuai dengan payload yang dikirim dari frontend untuk update.
// Sebelumnya hanya `Note`, tetapi UpdateByID di repository sudah diubah untuk menerima lebih banyak field.
type WorkScheduleUpdatePayload struct {
    Date           string `json:"date" validate:"required,datetime=2006-01-02"` // Tambahkan Date
    StartTime      string `json:"start_time" validate:"required,datetime=15:04"` // Tambahkan StartTime
    EndTime        string `json:"end_time" validate:"required,datetime=15:04"`     // Tambahkan EndTime
    Note           string `json:"note,omitempty"`
    RecurrenceRule string `json:"recurrence_rule,omitempty"` // Tambahkan RecurrenceRule
}


// --- DEFINISI STRUCT HOLIDAY BARU ---
// Holiday merepresentasikan struktur data hari libur nasional dari API eksternal.
// Digunakan untuk JSON marshalling dan sebagai tipe data di handler.
type Holiday struct {
	Date string `json:"Date"` // Field ini harus diawali huruf kapital agar bisa di-encode ke JSON
	Name string `json:"Name"` // Field ini harus diawali huruf kapital agar bisa di-encode ke JSON
}