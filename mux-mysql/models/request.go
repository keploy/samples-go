// Package models contains the structs and models for request and response
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
	ID      string `json:"id"`
	Website string `json:"website"`
}
