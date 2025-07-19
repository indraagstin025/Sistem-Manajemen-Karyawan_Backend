package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Department struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}


type DepartmentCount struct {
	Department string `bson:"_id" json:"department"`
	Count      int64  `bson:"count" json:"count"`
}