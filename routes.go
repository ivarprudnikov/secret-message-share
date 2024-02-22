package main

import (
	"context"
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
const SESS_COOKIE = "_i_remember"
const SESS_CSRF_KEY = "csrf"
const SESS_USER_KEY = "user"

// contextKey is the type used to store the user in the context.
type contextKey int

// userKey is the key used to store the user in the context.
const userKey contextKey = 50

func init() {
	tmpl = template.Must(template.ParseFS(templatesFs, "web/*.tmpl"))
}

func AddRoutes(
	mux *http.ServeMux,
	sessions *sessions.CookieStore,
	messages *storage.MessageStore,
	users *storage.UserStore,
) {
	preReq := newAppMiddleware(sessions, users)
	mux.Handle("GET /accounts/login", preReq(loginPageHandler(sessions)))
	mux.Handle("POST /accounts/login", preReq(loginAccountHandler(sessions, users)))
	mux.Handle("GET /accounts/new", preReq(createAccountPageHandler(sessions)))
	mux.Handle("POST /accounts", preReq(createAccountHandler(sessions, users)))
	mux.Handle("GET /messages", preReq(hasAuth(listMsgHandler(messages))))
	mux.Handle("POST /messages", preReq(hasAuth(createMsgHandler(sessions, messages))))
	mux.Handle("GET /messages/new", preReq(hasAuth(createMsgPageHandler(sessions))))
	mux.Handle("GET /messages/{id}", preReq(showMsgHandler(sessions, messages)))
	mux.Handle("POST /messages/{id}", preReq(showMsgFullHandler(sessions, messages)))
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

func loginPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		tmpl.ExecuteTemplate(w, "account.login.tmpl", sess.Values)
	}
}

func loginAccountHandler(sessions *sessions.CookieStore, store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
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
		usr, err := store.GetUserWithPass(username, password)
		if err != nil {
			sendError(w, "failed to login", err)
			return
		}
		if usr == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sess.Values[SESS_USER_KEY] = username
		err = sess.Save(r, w)
		if err != nil {
			sendError(w, "failed to save session", err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func createAccountPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		tmpl.ExecuteTemplate(w, "account.create.tmpl", sess.Values)
	}
}

func createAccountHandler(sessions *sessions.CookieStore, store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
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

func createMsgPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		tmpl.ExecuteTemplate(w, "message.create.tmpl", sess.Values)
	}
}

func createMsgHandler(sessions *sessions.CookieStore, store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(MAX_FORM_SIZE)
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
			sendError(w, "invalid token", nil)
			return
		}
		payload := r.PostForm.Get("payload")
		if payload == "" {
			sendError(w, "payload is empty", nil)
			return
		}
		// TODO only auth user is allowed
		msg, err := store.AddMessage(payload, "someuser")
		if err != nil {
			sendError(w, "failed to store message", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.created.tmpl", msg)
	}
}

func showMsgHandler(sessions *sessions.CookieStore, store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		msg, err := store.GetMessage(id)
		sess, _ := sessions.Get(r, SESS_COOKIE)
		if err != nil {
			sendError(w, "failed to get a message", err)
			return
		}
		if msg == nil {
			send404(w)
			return
		}
		tmpl.ExecuteTemplate(w, "message.show.tmpl", map[string]interface{}{
			"message":     msg,
			SESS_CSRF_KEY: sess.Values[SESS_CSRF_KEY],
		})
	}
}

func showMsgFullHandler(sessions *sessions.CookieStore, store *storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := r.ParseForm()
		if err != nil {
			sendError(w, "failed to read request body parameters", err)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
			sendError(w, "invalid token", nil)
			return
		}
		pin := r.PostForm.Get("pin")
		if pin == "" {
			sendError(w, "pin is empty", nil)
			return
		}
		msg, err := store.GetFullMessage(id, pin)
		if err != nil {
			sendError(w, "failed to get a message", err)
			return
		}
		if msg == nil {
			send404(w)
			return
		}
		tmpl.ExecuteTemplate(w, "message.show.tmpl", map[string]interface{}{
			"message": msg,
		})
	}
}

// adds CSRF token to the session of the get requests
func newAppMiddleware(sessions *sessions.CookieStore, users *storage.UserStore) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			sess, _ := sessions.Get(r, SESS_COOKIE)

			if r.Method == "GET" {
				// setup CSRF token for pages
				t, err := storage.MakeToken()
				if err != nil {
					sendError(w, "failed to setup csrf", err)
					return
				}
				sess.Values[SESS_CSRF_KEY] = t
			}

			// check if user is set, if yes then add it to context
			_username := sess.Values[SESS_USER_KEY]
			if username, ok := _username.(string); ok {
				user, err := users.GetUser(username)
				if err != nil || user == nil {
					sess.Values[SESS_USER_KEY] = nil
				}
				var ctx = r.Context()
				*r = *r.WithContext(context.WithValue(ctx, userKey, user))
			}

			err := sess.Save(r, w)
			if err != nil {
				sendError(w, "failed to save session", err)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

func hasAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		user := ctx.Value(userKey)
		if user == nil {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
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
