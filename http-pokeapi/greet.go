package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"strings"
)

const greeting = `Welcome, brave Pokemon Trainer,
to the PokeAPIExplorer—your open gateway to every corner of the Pokemon universe,
where you can list every known location area,
zoom into specific habitats to see which wild Pokemon appear, and
summon full stat sheets for any Pokemon by name, all in your choice of plain text, XML, or HTML; 
so grab your PokeBalls, venture into the tall grass of HTTP requests, and
remember: in this API-driven adventure, data is the true prize—gotta fetch them all!`

func (cfg *apiconfig) Greet(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		accept := r.Header.Get("Accept")
		switch {
		case strings.Contains(accept, "application/xml"):
			format = "xml"
		case strings.Contains(accept, "text/html"):
			format = "html"
		default:
			format = "plain"
		}
	}

	switch format {
	case "xml":
		w.Header().Set("Content-Type", "application/xml")
		type xmlGreeting struct {
			XMLName xml.Name `xml:"greeting"`
			Message string   `xml:"message"`
		}
		if err := xml.NewEncoder(w).Encode(xmlGreeting{Message: greeting}); err != nil {
			log.Printf("encode xml greet: %v", err)
		}

	case "html":
		w.Header().Set("Content-Type", "text/html")
		htmlBody := `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Greeting</title></head>
<body><h1>` + greeting + `</h1></body>
</html>`
		if _, err := w.Write([]byte(htmlBody)); err != nil {
			log.Printf("write html greet: %v", err)
		}

	case "plain":
		fallthrough
	default:
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(greeting)); err != nil {
			log.Printf("write plain greet: %v", err)
		}
	}
}
