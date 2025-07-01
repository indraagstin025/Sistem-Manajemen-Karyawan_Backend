package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Salary struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID     primitive.ObjectID `json:"user_id" bson:"user_id,omitempty"`
	Month      string             `json:"month" bson:"month,omitempty"`
	BaseSalary float64            `json:"base_salary" bson:"base_salary,omitempty"`
	Bonus      float64            `json:"bonus" bson:"bonus,omitempty"`
	Deduction  float64            `json:"deduction" bson:"deduction,omitempty"`
	TotalPaid  float64            `json:"total_paid" bson:"total_paid,omitempty"`
	Status     string             `json:"status" bson:"status,omitempty"`
	PaidAt     *time.Time         `json:"paid_at" bson:"paid_at,omitempty"`
	Notes      string             `json:"notes" bson:"notes,omitempty"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

type SalaryCreatePayload struct {
	UserID     string  `json:"user_id" validate:"required"`
	Month      string  `json:"month" validate:"required,datetime=2006-01"`
	BaseSalary float64 `json:"base_salary" validate:"required,min=0"`
	Bonus      float64 `json:"bonus" validate:"min=0"`
	Deduction  float64 `json:"deduction" validate:"min=0"`
	Status     string  `json:"status" validate:"required,oneof=paid unpaid"`
	Notes      string  `json:"notes"`
}

type SalaryUpdatePayload struct {
	BaseSalary *float64 `json:"base_salary,omitempty" validate:"omitempty,min=0"`
	Bonus      *float64 `json:"bonus,omitempty" validate:"omitempty,min=0"`
	Deduction  *float64 `json:"deduction,omitempty" validate:"omitempty,min=0"`
	Status     string   `json:"status,omitempty" validate:"omitempty,oneof=paid unpaid"`
	Notes      string   `json:"notes,omitempty"`
}
