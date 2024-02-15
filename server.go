package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/ivarprudnikov/secretshare/internal/storage"
)

const salt = "12345678123456781234567812345678"

func NewHttpHandler(messages *storage.MessageStore, users *storage.UserStore) http.Handler {
	mux := http.NewServeMux()
	AddRoutes(mux, messages, users)
	return mux
}

// main starts the server
func main() {
	messages := storage.NewMessageStore(salt)
	users := storage.NewUserStore(salt)
	handler := NewHttpHandler(messages, users)
	port := getPort()
	listenAddr := ":" + port
	log.Printf("About to listen on %s. Go to http://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, handler))
}

// getPort returns the port of this function app
func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return val
	}
	return "8080"
}
