package main

import (
	"context"
	_ "embed"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/ivarprudnikov/secretshare/internal/configuration"
	"github.com/ivarprudnikov/secretshare/internal/storage"
	"github.com/ivarprudnikov/secretshare/internal/storage/aztablestore"
	"github.com/ivarprudnikov/secretshare/internal/storage/memstore"
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
	messages, users := getStorageImplementation(config)
	handler := NewHttpHandler(sessions, messages, users)
	port := getPort()
	listenAddr := "127.0.0.1:" + port
	log.Printf("About to listen on %s. Go to http://%s/", port, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, handler))
}

// getPort returns the port of this function app
// in Azure the environmental variable will tell the port to run on FUNCTIONS_CUSTOMHANDLER_PORT
func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return val
	}
	return "8080"
}

// Production environment needs to work with Azure Table Storage which is not
// available locally. Locally an in-memory implementation of storage is used.
func getStorageImplementation(config *configuration.ConfigReader) (storage.MessageStore, storage.UserStore) {
	var messages storage.MessageStore
	var users storage.UserStore

	if config.IsProd() {
		messages = aztablestore.NewAzMessageStore(config.GetStorageAccountName(), config.GetMessagesTableName(), config.GetSalt())
		users = aztablestore.NewAzUserStore(config.GetStorageAccountName(), config.GetUsersTableName(), config.GetSalt())
	} else {
		messages = memstore.NewMemMessageStore(config.GetSalt())
		users = memstore.NewMemUserStore(config.GetSalt())
		bootstrapTestData(messages, users)
	}
	return messages, users
}

func bootstrapTestData(messages storage.MessageStore, users storage.UserStore) {
	// add test users
	users.AddUser(context.Background(), "joe", "joe", []string{})
	users.AddUser(context.Background(), "alice", "alice", []string{})
	users.AddUser(context.Background(), "admin", "admin", []string{storage.PERMISSION_READ_STATS})

	// add a test message
	msg, err := messages.AddMessage(context.Background(), "foobar", "joe")
	if err != nil {
		panic("Unexpected error")
	}
	log.Printf("Generated PIN for test message %s", msg.Pin)
}
