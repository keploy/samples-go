package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/heyyakash/keploy-go-samples/db"
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
	id, err := db.Store.EnterWebsiteToDB(req.Link)
	if err != nil {
		log.Printf("Error ", err)
		helpers.SendResponse(w, http.StatusInternalServerError, "Some Error occured", "", false)
		return
	}
	link := "http://localhost:8080" + "/" + strconv.FormatInt(id, 10)
	helpers.SendResponse(w, http.StatusOK, "Converted", link, true)
}
