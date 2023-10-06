package responseStruct

import (
	structs "user-onboarding/struct"

	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type GetCreatorDetailsResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message"`
	Data    *structs.UserDetails `json:"userData"`
}

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}
type GetUserResponse struct {
	Status   string                              `json:"status"`
	Message  string                              `json:"message"`
	Response map[string]*dynamodb.AttributeValue `json:"response,omitempty"`
}
