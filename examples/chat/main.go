package main

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

//go:embed index.html
var indexFile embed.FS

func main() {
	logger := Slog()
	ctx := context.Background()

	app := NewApp(logger)
	manager := websocket_manager.New(ctx, websocket_manager.Config{
		PingFrequency:     5 * time.Second,
		PingTimeout:       5 * time.Second,
		PongTimeout:       10 * time.Second,
		ConnectionHandler: app,
		Upgrader:          &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		Logger:            logger,
		Middlewares:       []websocket_manager.Middleware{connectionIdMiddleware, getUsernameMiddleware(app)},
		ResponseHeader:    nil,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, indexFile, "index.html")
	})
	mux.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("username") == false || r.URL.Query().Get("username") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := manager.UpgradeAndRunAsync(w, r, websocket_manager.ContextValue{Key: "username", Val: r.URL.Query().Get("username")}); err != nil {
			logger.ErrorContext(ctx, "failed to upgrade connection", "error", err)
			return
		}
	}))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.ErrorContext(ctx, "failed to start server", "error", err)
	}
}
