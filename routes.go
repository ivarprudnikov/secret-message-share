package main

import (
	"embed"
	"html/template"

	"errors"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

//go:embed web
var templatesFs embed.FS

var tmpl *template.Template

const MAX_FORM_SIZE = int64(3 << 20) // 3 MB
const SESS_NAME = "any"
const SESS_USER = "user"
const SESS_CSRF = "csrf"

func init() {
	tmpl = template.Must(template.ParseFS(templatesFs, "web/*.tmpl"))
}

func AddRoutes(
	mux *http.ServeMux,
	sessions *sessions.CookieStore,
	messages *storage.MessageStore,
	users *storage.UserStore,
) {
	mux.HandleFunc("GET /accounts/login", loginAccountHandler(sessions, users))
	mux.HandleFunc("POST /accounts/login", loginAccountHandler(sessions, users))
	mux.HandleFunc("GET /accounts/new", createAccountHandler(users))
	mux.HandleFunc("POST /accounts", createAccountHandler(users))

	mux.HandleFunc("GET /messages", listMsgHandler(messages))
	mux.HandleFunc("POST /messages", createMsgHandler(messages))
	mux.HandleFunc("GET /messages/new", createMsgHandler(messages))
	mux.Handle("GET /messages/{id}", showMsgHandler(messages))
	mux.Handle("POST /messages/{id}", showMsgHandler(messages))
	mux.HandleFunc("GET /", indexHandler)
}

// indexHandler returns the main index page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		send404(w)
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
	w.Header().Set("Expires", "0")                                         // Proxies
	w.Header().Add("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "index.tmpl", nil)
}

func loginAccountHandler(sessions *sessions.CookieStore, store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_NAME)
		if r.Method == "GET" {
			t, err := storage.MakeToken()
			if err != nil {
				sendError(w, "failed to setup csrf", err)
				return
			}
			sess.Values[SESS_CSRF] = t
			err = sess.Save(r, w)
			if err != nil {
				sendError(w, "failed to setup csrf", err)
				return
			}
			tmpl.ExecuteTemplate(w, "account.login.tmpl", map[string]interface{}{
				SESS_CSRF: t,
			})
			return
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF] {
			sendError(w, "invalid token", nil)
			return
		}
		username := r.PostForm.Get("username")
		if username == "" {
			sendError(w, "username is empty", nil)
			return
		}
		password := r.PostForm.Get("password")
		if password == "" {
			sendError(w, "password is empty", nil)
			return
		}
		usr, err := store.GetUser(username, password)
		if err != nil {
			sendError(w, "failed to login", err)
			return
		}
		if usr == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sess.Values[SESS_USER] = username
		err = sess.Save(r, w)
		if err != nil {
			sendError(w, "failed to save session", err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func createAccountHandler(store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			tmpl.ExecuteTemplate(w, "account.create.tmpl", nil)
			return
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		username := r.PostForm.Get("username")
		if username == "" {
			sendError(w, "username is empty", nil)
			return
		}
		password := r.PostForm.Get("password")
		if password == "" {
			sendError(w, "password is empty", nil)
			return
		}
		password2 := r.PostForm.Get("password2")
		if password2 != password {
			sendError(w, "passwords do not match", nil)
			return
		}
		usr, err := store.AddUser(username, password)
		if err != nil {
			sendError(w, "failed to create account", err)
			return
		}
		tmpl.ExecuteTemplate(w, "account.created.tmpl", usr)
	}
}

func listMsgHandler(store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		messages, err := store.ListMessages()
		if err != nil {
			sendError(w, "failed to list messages", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.list.tmpl", messages)
	}
}

func createMsgHandler(store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			tmpl.ExecuteTemplate(w, "message.create.tmpl", nil)
			return
		}

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
		msg, err := store.AddMessage(payload, "someuser")
		if err != nil {
			sendError(w, "failed to store message", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.created.tmpl", msg)
	}
}

func showMsgHandler(store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var msg *storage.Message
		var err error
		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				sendError(w, "failed to read request body parameters", err)
				return
			}
			pin := r.PostForm.Get("pin")
			if pin == "" {
				sendError(w, "pin is empty", nil)
				return
			}
			msg, err = store.GetFullMessage(id, pin)
			if err != nil {
				sendError(w, "failed to get a message", err)
				return
			}
		}

		// if PIN was not successful then do the regular message
		// retrieval
		if msg == nil {
			msg, err = store.GetMessage(id)
		}
		if err != nil {
			sendError(w, "failed to get a message", err)
			return
		}
		if msg == nil {
			send404(w)
			return
		}
		tmpl.ExecuteTemplate(w, "message.show.tmpl", msg)
	}
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
	apiError := ApiError{
		Message: message,
		Error:   err.Error(),
	}
	w.WriteHeader(http.StatusBadRequest)
	tmpl.ExecuteTemplate(w, "400.tmpl", apiError)
}

func send404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	tmpl.ExecuteTemplate(w, "404.tmpl", nil)
}
