package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

// Define the XML structure
type SystemCounter struct {
	XMLName    xml.Name           `xml:"system-counter"`
	Identifier string             `xml:"identifier,attr"`
	Name       string             `xml:"name,attr"`
	Value      int                `xml:"value,attr"`
	AdminData  AdministrativeData `xml:"administrative-data"`
}

type AdministrativeData struct {
	CreatedAt     string `xml:"created-at,attr"`
	CreatedBy     string `xml:"created-by,attr"`
	ChangedAt     string `xml:"changed-at,attr"`
	ChangedBy     string `xml:"changed-by,attr"`
	Comment       string `xml:"comment,attr"`
	LogicalSystem string `xml:"logical-system,attr"`
}

func xmlHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("----- Incoming Request -----")
	fmt.Println("Method:", r.Method)
	fmt.Println("URL:", r.URL.String())
	fmt.Println("Headers:")
	for key, values := range r.Header {
		fmt.Printf("  %s: %s\n", key, values)
	}
	fmt.Println("----------------------------")
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK) // Ensure a valid status code

	// Create a valid XML response
	response := SystemCounter{
		Identifier: "MySystemCounter",
		Name:       "MySystemCounterName",
		Value:      1,
		AdminData: AdministrativeData{
			CreatedAt:     "2025-02-28T15:41:43Z",
			CreatedBy:     "seeadmin",
			ChangedAt:     "2025-02-28T15:41:43Z",
			ChangedBy:     "seeadmin",
			Comment:       "string",
			LogicalSystem: "0002",
		},
	}

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "Error generating XML", http.StatusInternalServerError)
		return
	}

	// Add XML declaration manually
	xmlWithHeader := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" + string(xmlData))

	w.Write([]byte(xmlWithHeader))
}

func main() {
	fmt.Println("Starting server on port :9090")
	http.HandleFunc("/xml", xmlHandler)
	http.ListenAndServe(":9090", nil)
}
