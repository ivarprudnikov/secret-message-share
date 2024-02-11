package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/ivarprudnikov/secretshare/internal/storage"
)

// main starts the server
func main() {
	messageStore := storage.NewStore()
	mux := http.NewServeMux()
	addRoutes(mux, messageStore)
	port := getPort()
	listenAddr := ":" + port
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

// getPort returns the port of this function app
func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return val
	}
	return "8080"
}
