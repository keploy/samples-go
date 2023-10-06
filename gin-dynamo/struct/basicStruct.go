package structs

var SourceType = map[int]string{
	0: "EMAIL",
	1: "PHONE",
}

type SignUp struct {
	CreationSource     string `json:"handleName" binding:"required"`
	Password           string `json:"password" binding:"required"`
	CreationSourceType string `json:"email" binding:"required"`
}

type UserDetails struct {
	CreationSource     string `json:"creationSource,omitempty"`
	CreationSourceType string `json:"creationSourceType" binding:"required"`
	Password           string `json:"password,omitempty" binding:"required"`
	Email              string `json:"email,omitempty"`
	Phone              string `json:"phone,omitempty"`
	SocialHandles      string `json:"socialHandles,omitempty"`
	Bio                string `json:"bio,omitempty"`
	BankDetails        string `json:"bankDetails,omitempty"`
	Skills             string `json:"skills,omitempty"`
}
