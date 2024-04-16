package main

import (
	_ "embed"
	"log"
	"log/slog"
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
	// Setup a default logger and the level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	config := configuration.NewConfigReader()
	if valid, vars := config.IsValid(); !valid {
		log.Fatalf("Invalid config: %v", vars)
	}
	sessions := sessions.NewCookieStore([]byte(config.GetCookieAuth()), []byte(config.GetCookieEnc()))
	messages := storage.NewMemMessageStore(config.GetSalt())
	users := storage.NewMemUserStore(config.GetSalt())
	if !config.IsProd() {
		// add test users
		users.AddUser("joe", "joe")
		users.AddUser("alice", "alice")
		// add a test message
		messages.AddMessage("foobar", "joe")
	}
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
