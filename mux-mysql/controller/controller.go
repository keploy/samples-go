// Package controller contains the controller functions
package controller

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/db"
	"github.com/heyyakash/keploy-go-samples/helpers"
	"github.com/heyyakash/keploy-go-samples/models"
)

func CreateLink(store *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Print("Error decoding json", err)
			helpers.SendResponse(w, http.StatusBadRequest, "Error decoding JSON", "", false)
			return
		}
		if valid := helpers.CheckValidURL(req.Link); !valid {
			helpers.SendResponse(w, http.StatusBadRequest, "Enter Valid url (starting with 'http:// or https://')", "", false)
			return
		}
		id, err := db.EnterWebsiteToDB(req.Link, store)
		if err != nil {
			log.Print("Error ", err)
			helpers.SendResponse(w, http.StatusInternalServerError, err.Error(), "", false)
			return
		}
		link := "http://localhost:8080" + "/links/" + strconv.FormatInt(id, 10)
		helpers.SendResponse(w, http.StatusOK, "Converted", link, true)
	}
}

func RedirectUser(store *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		log.Printf("%v", id)
		link, err := db.GetWebsiteFromID(id, store)
		if err != nil {
			log.Print("Error ", err)
			helpers.SendResponse(w, http.StatusNotFound, "Website not found", "", false)
			return
		}
		http.Redirect(w, r, link, http.StatusTemporaryRedirect)
	}
}

func GetAllLinksFromWebsite(store *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		array, err := db.GetAllLinks(store)
		if err != nil {
			log.Print("Error ", err)
			helpers.SendResponse(w, http.StatusNotFound, "Some Error Retrieving the data"+err.Error(), "", false)
			return
		}
		helpers.SendGetResponse(w, array, http.StatusOK, true)
	}
}
