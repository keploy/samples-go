// // Package main sets up an HTTP server to interact with the Pokémon API via the pokeapi client.
package main

import (
	"http-pokeapi/internal/pokeapi"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
)

type apiconfig struct {
	client pokeapi.Client
}

func main() {
	time.Sleep(2 * time.Second)
	cfg := &apiconfig{
		client: pokeapi.Client{},
	}
	port := "8080"
	defer cfg.client.CloseIdleConnections()

	r := chi.NewRouter()
	s := chi.NewRouter()

	r.Mount("/api", s)

	s.Get("/locations", cfg.FetchLocations)
	s.Get("/locations/{location}", cfg.FetchPokemons)
	s.Get("/pokemon/{name}", cfg.AboutPokemon)

	// returns response in different formats based on query parameter or Accept header
	s.Get("/greet", cfg.Greet)

	servmux := corsmiddleware(r)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: servmux,
	}
	log.Printf("The server is live on port %s\n", port)
	err := srv.ListenAndServe()
	if err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}
