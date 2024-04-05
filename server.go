package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/ivarprudnikov/secretshare/internal/configuration"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

func NewHttpHandler(sessions *sessions.CookieStore, messages storage.MessageStore, users storage.UserStore) http.Handler {
	mux := http.NewServeMux()
	AddRoutes(mux, sessions, messages, users)
	return mux
}

// main starts the server
func main() {
	config := configuration.NewConfigReader()
	sessions := sessions.NewCookieStore([]byte(config.GetCookieAuth()), []byte(config.GetCookieEnc()))
	messages := storage.NewMemMessageStore(config.GetSalt())
	users := storage.NewMemUserStore(config.GetSalt())
	handler := NewHttpHandler(sessions, messages, users)
	port := getPort()
	listenAddr := "127.0.0.1:" + port
	log.Printf("About to listen on %s. Go to http://%s/", port, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, handler))
}

// getPort returns the port of this function app
func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return val
	}
	return "8080"
}
