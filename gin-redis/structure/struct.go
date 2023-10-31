package structure

import (
	"mime/multipart"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Influencer struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name     string             `json:"name,omitempty" bson:"name,omitempty"`
	LinkedIn string             `json:"linkedin,omitempty" bson:"linkedin,omitempty"`
	Company  string             `json:"company,omitempty" bson:"company,omitempty"`
}

type InfluencerFeedback struct {
	Name       string
	LinkedIn   string
	Post       *multipart.FileHeader
	Company    string
	Feedback   string
	Influencer interface{}
	ImageLink  string
}

type OTPData struct {
	OTP      int    `json:"otp"`
	UserName string `json:"username"`
}
