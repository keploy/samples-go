package models

type Request struct {
	Link string `json:"link"`
}

type Response struct {
	Message string `json:"message"`
	Link    string `json:"link"`
	Status  bool   `json:"status"`
}
