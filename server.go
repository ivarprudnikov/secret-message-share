package main

import (
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

//go:embed web/index.html
var indexHtml string

const MAX_FORM_SIZE = int64(3 << 20) // 3 MB

var messages = sync.Map{}

type Message struct {
	Content string    `json:"content,omitempty"`
	Digest  string    `json:"digest"`
	Pin     string    `json:"pin,omitempty"`
	Created time.Time `json:"created"`
}

// indexHandler returns the main index page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
	w.Header().Set("Expires", "0")                                         // Proxies
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprint(w, indexHtml)
}

func listMsgHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msgs []Message
	messages.Range(func(k, v any) bool {
		if msg, ok := v.(Message); ok {
			// do not expose sensitive info
			msgs = append(msgs, Message{
				Digest:  msg.Digest,
				Created: msg.Created,
			})
		} else {
			log.Printf("not a message %s %v", k, v)
		}
		return true
	})
	if len(msgs) < 1 {
		sendError(w, "no messages found", nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	marshalled, err := json.Marshal(msgs)
	if err != nil {
		sendError(w, "failed to render messages", err)
		return
	}
	fmt.Fprint(w, string(marshalled))
}

func createMsgHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseMultipartForm(MAX_FORM_SIZE)
	if err != nil {
		sendError(w, "failed to read request body parameters", err)
		return
	}
	payload := r.PostForm.Get("payload")
	if payload == "" {
		sendError(w, "payload is empty", nil)
		return
	}
	payloadHash := sha256.Sum256([]byte(payload))
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	pin, err := makePin()
	if err != nil {
		sendError(w, "failed to generate PIN, try again", nil)
		return
	}
	msg := Message{
		Content: payload,
		Digest:  payloadHashHex,
		Created: time.Now(),
		Pin:     fmt.Sprintf("%d", pin),
	}
	messages.Store(payloadHashHex, msg)

	w.Header().Add("Content-Type", "application/json")
	marshalled, err := json.Marshal(msg)
	if err != nil {
		sendError(w, "failed to render message", err)
		return
	}
	fmt.Fprint(w, string(marshalled))
}

// main starts the server
func main() {
	http.HandleFunc("/message/list", listMsgHandler)
	http.HandleFunc("/message/create", createMsgHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/index.html", indexHandler)
	port := getPort()
	listenAddr := ":" + port
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// simple pin generator
func makePin() (uint16, error) {
	c := 16
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

type ApiError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// sendError sends a json error response
func sendError(w http.ResponseWriter, message string, err error) {
	if err == nil {
		err = errors.New(message)
	}
	log.Printf("%s: %+v", message, err)
	w.Header().Set("Content-Type", "application/json")
	apiError := ApiError{
		Message: message,
		Error:   err.Error(),
	}
	apiErrorJson, marshalErr := json.Marshal(apiError)
	if marshalErr != nil {
		log.Fatalf("failed to marshal error: %+v", marshalErr)
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, string(apiErrorJson))
}

// getPort returns the port of this function app
func getPort() string {
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		return val
	}
	return "8080"
}
