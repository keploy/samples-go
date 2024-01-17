package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/heyyakash/keploy-go-samples/helpers"
	"github.com/heyyakash/keploy-go-samples/models"
)

func CreateLink(w http.ResponseWriter, r *http.Request) {
	var req models.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Print("Error decoding json", err)
		helpers.SendResponse(w, http.StatusBadRequest, "Error decoding JSON", "", false)
		return
	}
	helpers.SendResponse(w, http.StatusOK, "Got data", "", true)
}
