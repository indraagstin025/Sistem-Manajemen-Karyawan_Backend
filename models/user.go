package models // Pastikan nama paket ini sesuai dengan nama folder 'models' Anda

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name         string             `json:"name" bson:"name,omitempty"`
	Email        string             `json:"email" bson:"email,omitempty"`
	Password     string             `json:"password" bson:"password,omitempty"`
	Role         string             `json:"role" bson:"role,omitempty"`
	Position     string             `json:"position" bson:"position,omitempty"`
	Department   string             `json:"department" bson:"department,omitempty"`
	BaseSalary   float64            `json:"base_salary" bson:"base_salary,omitempty"`
	Address      string             `json:"address" bson:"address,omitempty"`     
	Photo        string             `json:"photo" bson:"photo,omitempty"`
	IsFirstLogin bool               `json:"is_first_login" bson:"isFirstLogin,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at,omitempty"`
}

type UserRegisterPayload struct {
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Email       string   `json:"email" validate:"omitempty,email"`
	Password    string  `json:"password" validate:"required,min=8,max=50,hasuppercase"`
	Role        string  `json:"role" validate:"required,oneof=admin karyawan"`
	Position    string  `json:"position"`
	Department  string  `json:"department"`
	BaseSalary  float64 `json:"base_salary" validate:"min=0"`
	Address     string  `json:"address" validate:"omitempty,min=5,max=255"` 
	Photo       string  `json:"photo" validate:"omitempty,url"` 
}

type UserLoginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserUpdatePayload struct {
	Name        string  `json:"name,omitempty"`
	Email       string  `json:"email,omitempty" validate:"omitempty,email"`
	Position    string  `json:"position,omitempty"`
	Department  string  `json:"department,omitempty"`
	BaseSalary  float64 `json:"base_salary,omitempty" validate:"omitempty,min=0"`
	Address     string  `json:"address,omitempty" validate:"omitempty,min=5,max=255"` 
	Photo       string  `json:"photo,omitempty" validate:"omitempty,url"`  
}

type ChangePasswordPayload struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=50,hasuppercase"`
}


