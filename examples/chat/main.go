package main

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

//go:embed index.html
var indexFile embed.FS

func main() {
	logger := Slog()
	ctx := context.Background()
	gCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := NewApp(logger)
	manager := websocket_manager.New(ctx, websocket_manager.Config{
		PingFrequency:     5 * time.Second,
		PingTimeout:       5 * time.Second,
		PongTimeout:       10 * time.Second,
		ConnectionHandler: app,
		Upgrader:          &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		OnError: func(msg string, err error) {
			logger.ErrorContext(ctx, msg, "error", err)
		},
		Middlewares:    []websocket_manager.Middleware{connectionIdMiddleware, getUsernameMiddleware(app)},
		ResponseHeader: nil,
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
	go func() {
		logger.InfoContext(ctx, "starting server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "failed to start server", "error", err)
		}
	}()

	<-gCtx.Done()

	logger.InfoContext(ctx, "shutting down")

	tCtx, tCancel := context.WithTimeout(ctx, 3*time.Second)
	defer tCancel()

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	wg.Go(func() {
		if err := server.Shutdown(tCtx); err != nil {
			logger.ErrorContext(tCtx, "failed to shutdown http server", "error", err)
		}
	})
	wg.Go(func() {
		if err := app.Shutdown(tCtx); err != nil {
			logger.ErrorContext(tCtx, "failed to shutdown app", "error", err)
		}
	})
}
