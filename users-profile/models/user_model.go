// Package models contain the models to represent our application data
package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User struct defines the user with required properties.
// Added omitempty and validate:"required" to the struct tag to tell Gin-gonic to ignore empty fields and make the field required, respectively.
type User struct {
	ID          primitive.ObjectID `json:"id,omitempty"`
	Username    string             `json:"username,omitempty" validate:"required"`
	Name        string             `json:"name,omitempty" validate:"required"`
	Nationality string             `json:"nationality,omitempty" validate:"required"`
	Title       string             `json:"title,omitempty" validate:"required"`
	Hobbies     string             `json:"hobbies,omitempty" validate:"required"`
	Linkedin    string             `json:"linkedin,omitempty" validate:"required"`
	Twitter     string             `json:"twitter,omitempty" validate:"required"`
}
