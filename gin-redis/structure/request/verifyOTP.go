package requestStruct

type OTPRequest struct {
	OTP   int    `json:"otp" binding:"required"`
	Email string `json:"email" binding:"required"`
}
