package main

import "net/http"

type locationres struct {
	Location []string `json:"location"`
}

func (cfg *apiconfig) FetchLocations(w http.ResponseWriter, r *http.Request) {
	res, err := cfg.client.LocationArearesponse()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var loc []string

	for _, location := range res.Results {
		loc = append(loc, location.Name)
	}

	respondWithJson(w, http.StatusOK, locationres{
		Location: loc,
	})

}
