package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
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

var upgrader = &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
	return true
}}

func main() {
	logger := Slog()
	ctx := context.Background()
	gCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	bus := NewBus()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, indexFile, "index.html")
	})
	mux.HandleFunc("/active", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("%d", bus.ActiveCount())))
	})
	mux.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("username") == false || r.URL.Query().Get("username") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		username := r.URL.Query().Get("username")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.ErrorContext(ctx, "failed to upgrade connection", "error", err)
			return
		}

		go func() {
			err := websocket_manager.Run(conn, websocket_manager.SocketCreatorFunc(func() (websocket_manager.Socket, error) {
				return NewClient(context.WithValue(ctx, "username", username), logger, username, bus), nil
			}), &websocket_manager.Config{
				PingMessage:   websocket_manager.PingMessage(nil),
				PingFrequency: 3 * time.Second,
				PongTimeout:   7 * time.Second,
				WriteTimeout:  3 * time.Second,
			})
			if err != nil {
				logger.ErrorContext(ctx, "websocket connection failed", "error", err)
				return
			}
		}()
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
		if err := bus.Shutdown(tCtx); err != nil {
			logger.ErrorContext(tCtx, "failed to shutdown app", "error", err)
		}
	})
}
