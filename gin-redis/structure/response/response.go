// Package responsestruct contains struct for api response
package responsestruct

type SuccessResponse struct {
	Status   string `json:"status,omitempty"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
	OTP      string `json:"otp,omitempty"`
}
