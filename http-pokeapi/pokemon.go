package main

import (
	"net/http"

	"github.com/go-chi/chi"
)

type PokemonNamesres struct {
	Names []string `json:"names"`
}
type pokemonstats struct {
	Name     string `json:"name"`
	BaseStat int    `json:"basestat"`
}

type Pokemonres struct {
	Name   string         `json:"name"`
	Height int            `json:"height"`
	Weight int            `json:"weight"`
	Stats  []pokemonstats `json:"stats"`
	Types  []string       `json:"types"`
}

func (cfg *apiconfig) FetchPokemons(w http.ResponseWriter, r *http.Request) {
	location := chi.URLParam(r, "location")

	res, err := cfg.client.Pokelocationres(location)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	var pokemon []string

	for _, poke := range res.PokemonEncounters {
		pokemon = append(pokemon, poke.Pokemon.Name)
	}

	respondWithJson(w, http.StatusOK, pokemon)

}

func (cfg *apiconfig) AboutPokemon(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	res, err := cfg.client.Pokemon(name)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	var statsresponse []pokemonstats

	for _, stats := range res.Stats {
		statsresponse = append(statsresponse, pokemonstats{
			Name:     stats.Stat.Name,
			BaseStat: stats.BaseStat,
		})
	}

	var restype []string

	for _, t := range res.Types {
		restype = append(restype, t.Type.Name)
	}

	respondWithJson(w, http.StatusOK, Pokemonres{
		Name:   res.Name,
		Height: res.Height,
		Weight: res.Weight,
		Stats:  statsresponse,
		Types:  restype,
	})
}
