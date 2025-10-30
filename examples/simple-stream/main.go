package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

//go:embed index.html
var indexFile embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	pool := &ConnectionPool{}
	defer pool.UnregisterAll()
	manager := websocket_manager.New(ctx, websocket_manager.Config{
		WriteControlTimeout: 5 * time.Second,
		PingFrequency:       5 * time.Second,
		PingTimeout:         5 * time.Second,
		PongTimeout:         10 * time.Second,
		MessageWriter:       getMessageWriter(logger, pool),
		MessageReader:       getMessageReader(logger, pool),
		Upgrader:            &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		Logger:              logger,
		Middlewares:         []websocket_manager.Middleware{connectionIdMiddleware, getUsernameMiddleware(pool)},
		ResponseHeader:      nil,
		OnConnect: websocket_manager.ConnectionEventHandlerFunc(func(ctx context.Context) {
			logger.InfoContext(ctx, "connected", "connectionId", ctx.Value("connectionId").(string))
		}),
		OnDisconnect: websocket_manager.ConnectionEventHandlerFunc(func(ctx context.Context) {
			logger.InfoContext(ctx, "disconnected", "connectionId", ctx.Value("connectionId").(string))
		}),
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

		worker, err := manager.Upgrade(w, r, websocket_manager.ContextValue{Key: "username", Val: r.URL.Query().Get("username")})
		if err != nil {
			logger.ErrorContext(ctx, "failed to upgrade connection", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		go func() { // Run the websocket worker in a goroutine to free allocated memory as soon as possible.
			if err := worker.Run(); err != nil {
				logger.ErrorContext(ctx, "worker failed", "error", err)
				return
			}
		}()
	}))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.ErrorContext(ctx, "failed to start server", "error", err)
	}
}
