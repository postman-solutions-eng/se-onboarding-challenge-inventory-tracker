// Command server runs the Inventory Tracker backend: a drop-in replacement
// for postman/mocks/inventory-tracker-mock/default.js that the existing
// Postman collection can run against unchanged.
package main

import (
	"log"
	"net/http"
	"os"

	"inventory-tracker-server/internal/httpapi"
	"inventory-tracker-server/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4500"
	}
	apiKey := os.Getenv("API_KEY")

	s := store.New()
	api := httpapi.New(s, apiKey)

	log.Printf("inventory-tracker-server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, api.Routes()); err != nil {
		log.Fatal(err)
	}
}
