package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
	if valid := helpers.CheckValidURL(req.Link); valid == false {
		helpers.SendResponse(w, http.StatusBadRequest, "Enter Valid url (starting with 'http:// or https://')", "", false)
		return
	}
	id, err := db.Store.EnterWebsiteToDB(req.Link)
	if err != nil {
		log.Print("Error ", err)
		helpers.SendResponse(w, http.StatusInternalServerError, "Some Error occured", "", false)
		return
	}
	link := "http://localhost:8080" + "/" + strconv.FormatInt(id, 10)
	helpers.SendResponse(w, http.StatusOK, "Converted", link, true)
}

func RedirectUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	log.Printf("%v", id)
	link, err := db.Store.GetWebsiteFromId(id)
	if err != nil {
		log.Print("Error ", err)
		helpers.SendResponse(w, http.StatusNotFound, "Website not found", "", false)
		return
	}
	http.Redirect(w, r, link, http.StatusTemporaryRedirect)
}

func GetAllLinksFromWebsite(w http.ResponseWriter, r *http.Request) {
	array, err := db.Store.GetAllLinks()
	if err != nil {
		log.Print("Error ", err)
		helpers.SendResponse(w, http.StatusNotFound, "Some Error Retrieving the data", "", false)
		return
	}
	helpers.SendGetResponse(w, array, http.StatusOK, true)
}
