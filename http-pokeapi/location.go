// Package main provides handlers for location-related API endpoints.
package main

import "net/http"

type locationres struct {
	Location []string `json:"location"`
}

func (cfg *apiconfig) FetchLocations(w http.ResponseWriter, _ *http.Request) {
	res, err := cfg.client.LocationArearesponse()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var loc []string

	for _, location := range res.Results {
		loc = append(loc, location.Name)
	}

	respondWithJSON(w, http.StatusOK, locationres{
		Location: loc,
	})

}
