package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"regexp"

	"errors"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

const MAX_FORM_SIZE = int64(3 << 20) // 3 MB
const SESS_COOKIE = "_i_remember"
const SESS_CSRF_KEY = "csrf"
const SESS_USER_KEY = "user"
const VIEW_SESS_KEY = "session"
const VIEW_DATA_KEY = "data"
const VIEW_ERROR_KEY = "error"
const failedPathQueryKey = "failedPath"

// contextKey is the type used to store the user in the context.
type contextKey int

// userKey is the key used to store the user in the context.
const userKey contextKey = 50

// templates get embedded in the binary
//
//go:embed web
var templatesFs embed.FS

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.ParseFS(templatesFs, "web/*.tmpl"))
}

func AddRoutes(
	mux *http.ServeMux,
	sessions *sessions.CookieStore,
	messages storage.MessageStore,
	users storage.UserStore,
) {
	preReq := newAppMiddleware(sessions, users)
	mux.Handle("GET /accounts/login", preReq(loginPageHandler(sessions)))
	mux.Handle("POST /accounts/login", preReq(loginAccountHandler(sessions, users)))
	mux.Handle("GET /accounts/logout", preReq(logoutAccountHandler(sessions)))
	mux.Handle("GET /accounts/new", preReq(createAccountPageHandler(sessions)))
	mux.Handle("POST /accounts", preReq(createAccountHandler(sessions, users)))
	mux.Handle("GET /messages", preReq(hasAuth(listMsgHandler(sessions, messages))))
	mux.Handle("POST /messages", preReq(hasAuth(createMsgHandler(sessions, messages))))
	mux.Handle("GET /messages/new", preReq(hasAuth(createMsgPageHandler(sessions))))
	mux.Handle("GET /messages/{id}", preReq(showMsgHandler(sessions, messages)))
	mux.Handle("POST /messages/{id}", preReq(showMsgFullHandler(sessions, messages)))
	mux.Handle("GET /stats", preReq(hasAuth(hasPermission(storage.PERMISSION_READ_STATS, statsHandler(sessions, users, messages)))))
	mux.Handle("GET /", indexPageHandler(sessions))
}

// indexHandler returns the main index page
func indexPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			send404(w)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
		w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
		w.Header().Set("Expires", "0")                                         // Proxies
		w.Header().Add("Content-Type", "text/html")
		tmpl.ExecuteTemplate(w, "index.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

func loginPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		redirectPath := ""
		failedPath := r.URL.Query().Get(failedPathQueryKey)
		if failedPath != "" {
			slog.LogAttrs(r.Context(), slog.LevelInfo, "needs to redirect to protected path")
			parsedFailedPath, err := url.Parse(failedPath)
			if err == nil {
				redirectPath = parsedFailedPath.Path
			}
		}
		tmpl.ExecuteTemplate(w, "account.login.tmpl", map[string]interface{}{
			VIEW_SESS_KEY:      sess.Values,
			failedPathQueryKey: redirectPath,
		})
	}
}

func loginAccountHandler(sessions *sessions.CookieStore, store storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		err := r.ParseForm()
		if err != nil {
			sendError(r.Context(), sess, w, "failed to read request body parameters", err)
			return
		}
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
			sendError(r.Context(), sess, w, "invalid csrf token", nil)
			return
		}
		username := r.PostForm.Get("username")
		if username == "" {
			sendError(r.Context(), sess, w, "username is empty", nil)
			return
		}
		password := r.PostForm.Get("password")
		if password == "" {
			sendError(r.Context(), sess, w, "password is empty", nil)
			return
		}
		usr, err := store.GetUserWithPass(r.Context(), username, password)
		if err != nil {
			sendError(r.Context(), sess, w, "failed to login", err)
			return
		}
		if usr == nil {
			slog.LogAttrs(r.Context(), slog.LevelInfo, "user not found with username/pass", slog.String("username", username))
			sendError(r.Context(), sess, w, "failed to login", err)
			return
		}
		sess.Values[SESS_USER_KEY] = username
		err = sess.Save(r, w)
		if err != nil {
			sendError(r.Context(), sess, w, "failed to save session", err)
			return
		}

		redirectPath := "/"
		failedPath := r.PostForm.Get(failedPathQueryKey)
		if failedPath != "" {
			slog.LogAttrs(r.Context(), slog.LevelInfo, "needs to redirect to protected path")
			parsedFailedPath, err := url.Parse(failedPath)
			if err == nil {
				redirectPath = parsedFailedPath.Path
			}
		}

		slog.LogAttrs(r.Context(), slog.LevelInfo, "user successfully logged in, redirecting", slog.String("username", username), slog.String("path", redirectPath))
		http.Redirect(w, r, redirectPath, http.StatusSeeOther)
	}
}

func logoutAccountHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		sess.Values[SESS_USER_KEY] = nil
		err := sess.Save(r, w)
		if err != nil {
			sendError(r.Context(), sess, w, "failed to save session", err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func createAccountPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		tmpl.ExecuteTemplate(w, "account.create.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

func createAccountHandler(sessions *sessions.CookieStore, store storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		err := r.ParseForm()
		if err != nil {
			sendError(r.Context(), sess, w, "failed to read request body parameters", err)
			return
		}
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
			sendError(r.Context(), sess, w, "invalid token", nil)
			return
		}
		username := r.PostForm.Get("username")
		if username == "" {
			sendError(r.Context(), sess, w, "username is empty", nil)
			return
		}
		if matched, err := regexp.MatchString(`^[A-Za-z0-9_-]+$`, username); err != nil || !matched {
			sendError(r.Context(), sess, w, "username can only consist of letters, numbers, underscore (_) and dash (-)", nil)
			return
		}
		password := r.PostForm.Get("password")
		if password == "" {
			sendError(r.Context(), sess, w, "password is empty", nil)
			return
		}
		password2 := r.PostForm.Get("password2")
		if password2 != password {
			sendError(r.Context(), sess, w, "passwords do not match", nil)
			return
		}
		usr, err := store.AddUser(r.Context(), username, password, []string{})
		if err != nil {
			sendError(r.Context(), sess, w, "failed to create account", err)
			return
		}
		tmpl.ExecuteTemplate(w, "account.created.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
			VIEW_DATA_KEY: usr,
		})
	}
}

func listMsgHandler(sessions *sessions.CookieStore, store storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		sess, _ := sessions.Get(r, SESS_COOKIE)
		username := sess.Values[SESS_USER_KEY]
		messages, err := store.ListMessages(r.Context(), username.(string))
		if err != nil {
			sendError(r.Context(), sess, w, "failed to list messages", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.list.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
			VIEW_DATA_KEY: messages,
		})
	}
}

func createMsgPageHandler(sessions *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		tmpl.ExecuteTemplate(w, "message.create.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

func createMsgHandler(sessions *sessions.CookieStore, store storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		err := r.ParseMultipartForm(MAX_FORM_SIZE)
		if err != nil {
			sendError(r.Context(), sess, w, "failed to read request body parameters", err)
			return
		}
		csrf := r.PostForm.Get("_csrf")
		if csrf == "" || csrf != sess.Values[SESS_CSRF_KEY] {
			sendError(r.Context(), sess, w, "invalid token", nil)
			return
		}
		payload := r.PostForm.Get("payload")
		if payload == "" {
			sendError(r.Context(), sess, w, "payload is empty", nil)
			return
		}
		username := sess.Values[SESS_USER_KEY]
		msg, err := store.AddMessage(r.Context(), payload, username.(string))
		if err != nil {
			sendError(r.Context(), sess, w, "failed to store message", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.created.tmpl", map[string]interface{}{
			VIEW_SESS_KEY: sess.Values,
			VIEW_DATA_KEY: msg,
		})
	}
}

func showMsgHandler(sessions *sessions.CookieStore, store storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		msg, err := store.GetMessage(r.Context(), id)
		sess, _ := sessions.Get(r, SESS_COOKIE)
		if err != nil {
			sendError(r.Context(), sess, w, "failed to get a message", err)
			return
		}
		if msg == nil {
			send404(w)
			return
		}
		tmpl.ExecuteTemplate(w, "message.show.tmpl", map[string]interface{}{
			VIEW_DATA_KEY: msg,
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

func showMsgFullHandler(sessions *sessions.CookieStore, store storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sess, _ := sessions.Get(r, SESS_COOKIE)
		err := r.ParseForm()
		if err != nil {
			slog.LogAttrs(r.Context(), slog.LevelError, "failed to read request body parameters")
			sendError(r.Context(), sess, w, "failed to get a message", err)
			return
		}
		csrf := r.PostForm.Get("_csrf")
		csrfReal := sess.Values[SESS_CSRF_KEY].(string)
		slog.LogAttrs(r.Context(), slog.LevelDebug, "csrf token in the session", slog.String("csrf", csrfReal))
		slog.LogAttrs(r.Context(), slog.LevelDebug, "csrf token in the post form", slog.String("csrf", csrf))
		if csrf == "" || csrf != csrfReal {
			slog.LogAttrs(r.Context(), slog.LevelError, "invalid csrf token")
			sendError(r.Context(), sess, w, "failed to get a message", nil)
			return
		}
		pin := r.PostForm.Get("pin")
		if pin == "" {
			slog.LogAttrs(r.Context(), slog.LevelError, "message pin is empty")
			sendError(r.Context(), sess, w, "failed to get a message", nil)
			return
		}
		msg, err := store.GetFullMessage(r.Context(), id, pin)
		if err != nil || msg == nil {
			sendError(r.Context(), sess, w, "failed to get a message", err)
			return
		}
		tmpl.ExecuteTemplate(w, "message.show.tmpl", map[string]interface{}{
			VIEW_DATA_KEY: msg,
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

func statsHandler(sessions *sessions.CookieStore, userStore storage.UserStore, messageStore storage.MessageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessions.Get(r, SESS_COOKIE)
		users, err := userStore.CountUsers(r.Context())
		if err != nil {
			sendError(r.Context(), sess, w, "failed to get a user count", err)
			return
		}
		messages, err := messageStore.CountMessages(r.Context())
		if err != nil {
			sendError(r.Context(), sess, w, "failed to get a message count", err)
			return
		}
		tmpl.ExecuteTemplate(w, "stats.tmpl", map[string]interface{}{
			VIEW_DATA_KEY: map[string]int64{
				"TotalUsers":    users,
				"TotalMessages": messages,
			},
			VIEW_SESS_KEY: sess.Values,
		})
	}
}

// Main app middleware handles the session cookie
// also, finds and adds the user to the context if the session is valid
// also, adds a CSRF token to the session of GET requests, to be used in forms
func newAppMiddleware(sessions *sessions.CookieStore, users storage.UserStore) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			sess, _ := sessions.Get(r, SESS_COOKIE)

			if r.Method == "GET" {
				// setup CSRF token for pages
				t, err := crypto.MakeToken()
				if err != nil {
					sendError(r.Context(), sess, w, "failed to setup csrf", err)
					return
				}
				sess.Values[SESS_CSRF_KEY] = t
				slog.LogAttrs(r.Context(), slog.LevelDebug, "csrf token generated", slog.String("csrf", t))
			}

			// check if user is set, if yes then add it to context
			_username := sess.Values[SESS_USER_KEY]
			if username, ok := _username.(string); ok {
				var ctx = r.Context()
				user, err := users.GetUser(r.Context(), username)
				if err != nil {
					slog.LogAttrs(ctx, slog.LevelError, "failed to find session user", slog.String("username", username))
				}
				if err != nil || user == nil {
					sess.Values[SESS_USER_KEY] = nil
				} else {
					slog.LogAttrs(ctx, slog.LevelInfo, "setting session user in context", slog.String("username", username))
					*r = *r.WithContext(context.WithValue(ctx, userKey, user))
				}
			}

			err := sess.Save(r, w)
			if err != nil {
				sendError(r.Context(), sess, w, "failed to save session", err)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

// authentication check expects the user to be set when the session cookie was parsed
func hasAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		user := ctx.Value(userKey)
		u, ok := user.(*storage.User)
		if user == nil || !ok {
			slog.LogAttrs(ctx, slog.LevelInfo, "user not set, redirecting to login", slog.String("path", r.URL.Path))
			http.Redirect(w, r, fmt.Sprintf("/accounts/login?%s=%s", failedPathQueryKey, r.URL.Path), http.StatusSeeOther)
			return
		}
		slog.LogAttrs(ctx, slog.LevelInfo, "user authenticated", slog.String("username", u.PartitionKey), slog.String("path", r.URL.Path))
		h.ServeHTTP(w, r)
	})
}

// This ought to be used after the authentication check (hasAuth)
// Additional check for user here is to avoid unexpected usage
func hasPermission(permission string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		user := ctx.Value(userKey)
		u, ok := user.(*storage.User)
		if !ok {
			slog.LogAttrs(ctx, slog.LevelInfo, "user unauthorized", slog.String("path", r.URL.Path))
			w.WriteHeader(http.StatusUnauthorized)
			tmpl.ExecuteTemplate(w, "401.tmpl", nil)
			return
		}
		if !u.HasPermission(permission) {
			slog.LogAttrs(ctx, slog.LevelInfo, "access forbidden", slog.String("username", u.PartitionKey), slog.String("path", r.URL.Path))
			w.WriteHeader(http.StatusForbidden)
			tmpl.ExecuteTemplate(w, "403.tmpl", nil)
			return
		}
		slog.LogAttrs(ctx, slog.LevelInfo, "user has access", slog.String("username", u.PartitionKey), slog.String("permission", permission), slog.String("path", r.URL.Path))
		h.ServeHTTP(w, r)
	})
}

type ApiError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// sendError sends a json error response and logs the error message
func sendError(ctx context.Context, sess *sessions.Session, w http.ResponseWriter, message string, err error) {
	if err == nil {
		err = errors.New(message)
	}
	slog.LogAttrs(ctx, slog.LevelError, "request processing failed", slog.String("message", message), slog.Any("error", err))
	apiError := ApiError{
		Message: message,
	}
	w.WriteHeader(http.StatusBadRequest)
	tmpl.ExecuteTemplate(w, "400.tmpl", map[string]interface{}{
		VIEW_SESS_KEY:  sess.Values,
		VIEW_ERROR_KEY: apiError,
	})
}

func send404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	tmpl.ExecuteTemplate(w, "404.tmpl", nil)
}
