package server

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"goths-demo/pkg"
	"goths-demo/sqlc/db"
	"goths-demo/templ"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func (srv *server) RootHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Valid() != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (srv *server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	homePage := templ.HomePage()
	homePage.Render(r.Context(), w)
}

func (srv *server) LoginGetHandler(w http.ResponseWriter, r *http.Request) {
	loginForm := templ.LoginForm("")
	loginForm.Render(r.Context(), w)
}

func (srv *server) LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := srv.queries.Authenticate(r.Context(), db.AuthenticateParams{
		Username: sql.NullString{
			String: username,
			Valid:  true,
		},
		Password: sql.NullString{
			String: password,
			Valid:  true,
		},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			templ.LoginForm("login failed").Render(r.Context(), w)
			return
		}

		slog.Error("Error during login", "error", err)
		templ.LoginForm("retry later").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: cookieName,

		// Do not do this in production. This is only for learning.
		Value: user.Username.String,

		MaxAge: int((5 * time.Minute).Seconds()),
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (srv *server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "",
		MaxAge: -1,
	})
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (srv *server) AddPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	values := pkg.AddPostFormValues{
		Post: r.FormValue("Post"),
	}

	errors := values.Validate()

	// If validation fails, return form with errors
	if len(errors) > 0 {
		templ.AddPostForm(values, errors, "").Render(r.Context(), w)
		return
	}

	// Get username from cookie (checkAuth middleware ensures this exists)
	cookie, _ := r.Cookie(cookieName)
	username := cookie.Value

	// Create the post using the updated query that returns post ID
	postID, err := srv.queries.AddPost(r.Context(), db.AddPostParams{
		Content:  sql.NullString{String: values.Post, Valid: true},
		Username: sql.NullString{String: username, Valid: true},
	})
	if err != nil {
		slog.Error("Error creating post", "error", err)
		systemErrors := map[string]string{
			"system": "Failed to create post. Please try again.",
		}
		templ.AddPostForm(values, systemErrors, "").Render(r.Context(), w)
		return
	}

	// Broadcast the new post to all WebSocket clients
	postMsg := PostMessage{
		Username: username,
		Content:  values.Post,
		PostID:   postID,
	}

	select {
	case srv.postBroadcast <- postMsg:
	default:
		slog.Warn("Post broadcast channel is full, post not broadcasted")
	}

	// Success - return fresh form with success message including post ID
	emptyValues := pkg.AddPostFormValues{}
	emptyErrors := map[string]string{}
	successMessage := fmt.Sprintf("Post created successfully! (ID: %d)", postID)
	templ.AddPostForm(emptyValues, emptyErrors, successMessage).Render(r.Context(), w)
}

func (srv *server) GetTimelineHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("WebSocket connection attempt", "remote_addr", r.RemoteAddr)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("WebSocket connection established", "remote_addr", r.RemoteAddr)

	// Create a client channel for this connection
	clientChan := make(chan PostMessage, 10)

	// Register this client
	srv.clientsMutex.Lock()
	srv.clients[clientChan] = true
	clientCount := len(srv.clients)
	srv.clientsMutex.Unlock()

	slog.Info("Client registered for WebSocket", "total_clients", clientCount)

	// Cleanup when connection closes
	defer func() {
		select {
		case srv.cleanupClients <- clientChan:
		default:
		}
	}()

	// Send initial state using templ template
	var buf bytes.Buffer
	if err := templ.TimelineInitial().Render(r.Context(), &buf); err != nil {
		slog.Error("Failed to render initial template", "error", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
		slog.Error("Failed to send initial message", "error", err)
		return
	}

	// Listen for new posts and send them to the WebSocket
	for post := range clientChan {
		slog.Info("Got post in the getTLHandler", "post", post)
		// Render the post swap template
		var postBuf bytes.Buffer
		swapTemplate := templ.TimelinePostSwap(post.Username, post.Content, post.PostID)
		if err := swapTemplate.Render(r.Context(), &postBuf); err != nil {
			slog.Error("Failed to render post swap template", "error", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, postBuf.Bytes()); err != nil {
			slog.Error("Failed to send post", "error", err)
			break
		}
	}
}

func checkAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil || cookie.Valid() != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}
