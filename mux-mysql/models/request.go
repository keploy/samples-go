package models

type Request struct {
	Link string `json:"link"`
}

type Response struct {
	Message string `json:"message"`
	Link    string `json:"link"`
	Status  bool   `json:"status"`
}
type GETResponse struct {
	Message interface{} `json:"message"`
	Status  bool        `json:"status"`
}

type Table struct {
	Id      string `json:"id"`
	Website string `json:"website"`
}
