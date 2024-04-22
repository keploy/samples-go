package helpers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/heyyakash/keploy-go-samples/models"
)

func SendResponse(w http.ResponseWriter, code int, message string, link string, status bool) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(&models.Response{Message: message, Link: link, Status: status}); err != nil {
		log.Fatalf("Error writing JSON: %v", err)
	}
}

func SendGetResponse(w http.ResponseWriter, data interface{}, status int, success bool) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(&models.GETResponse{Message: data, Status: success}); err != nil {
		log.Fatalf("Error writing JSON: %v", err)
	}
}
