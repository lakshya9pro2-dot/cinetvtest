package main

import (
	"log"
	"net/http"
	"os"

	"nanoserver/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "1937"
	}

	log.Printf("NanoServer running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, http.HandlerFunc(handler.Handler)); err != nil {
		log.Fatal(err)
	}
}
