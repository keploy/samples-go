package requestStruct

type GetUserDetailsRequest struct {
	Email string `json:"handleName"`
}

type UserLogin struct {
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
}
