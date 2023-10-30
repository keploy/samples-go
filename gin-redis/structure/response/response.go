package responseStruct

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message interface{} `json:"message"`
}
