package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type WorkScheduleCreatePayload struct {
	Date           string `json:"date" validate:"required,datetime=2006-01-02"`
	StartTime      string `json:"start_time" validate:"required,datetime=15:04"`
	EndTime        string `json:"end_time" validate:"required,datetime=15:04"`
	Note           string `json:"note"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"`
}


type WorkScheduleUpdatePayload struct {
    Date           string `json:"date" validate:"required,datetime=2006-01-02"` 
    StartTime      string `json:"start_time" validate:"required,datetime=15:04"` 
    EndTime        string `json:"end_time" validate:"required,datetime=15:04"`    
    Note           string `json:"note,omitempty"`
    RecurrenceRule string `json:"recurrence_rule,omitempty"`  
}



type Holiday struct {
	Date string `json:"Date"` 
	Name string `json:"Name"` 
}