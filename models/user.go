package models

import (
	"time"
)

type User struct {
	Name        string    `json:"name,omitempty" bson:"name,omitempty" `
	AvatarName  string    `json:"avatar_name,omitempty" bson:"avatar_name,omitempty" `
	AvatarType  string    `json:"avatar_type,omitempty" bson:"avatar_type,omitempty" `
	Age         int       `json:"age,omitempty" bson:"age,omitempty"`
	YearOfBirth int       `json:"year_of_birth,omitempty" bson:"year_of_birth,omitempty" `
	Note        string    `json:"note,omitempty" bson:"note,omitempty"`
	Email       string    `json:"email,omitempty" bson:"email,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}
