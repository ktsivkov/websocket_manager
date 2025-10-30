package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

func getUsernameFromContext(ctx context.Context) string {
	return ctx.Value("username").(string)
}

func NewApp(logger *slog.Logger) *App {
	return &App{
		logger: logger,
		pool:   make(map[string]chan []byte),
		mu:     &sync.RWMutex{},
	}
}

type App struct {
	logger *slog.Logger
	mu     *sync.RWMutex
	pool   map[string]chan []byte
}

func (a *App) OnConnect(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	{
		a.mu.Lock()
		defer a.mu.Unlock()
		a.pool[username] = make(chan []byte)
	}

	{
		a.mu.RLock()
		defer a.mu.RUnlock()
		a.logger.InfoContext(ctx, "new connection", "connectionId", ctx.Value("connectionId").(string), "username", username)
		data, err := json.Marshal(Message{Type: TypeUserConnected, Data: username})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to marshal message response", "error", err, "payload", string(data), "username", username)
			return
		}
		for _, ch := range a.pool {
			ch <- data
		}
	}
	a.logger.InfoContext(ctx, "all notified")
}

func (a *App) OnDisconnect(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	username := getUsernameFromContext(ctx)
	if ch, ok := a.pool[username]; ok {
		close(ch)
		delete(a.pool, getUsernameFromContext(ctx))
	}
	data, err := json.Marshal(Message{Type: TypeUserDisconnected, Data: username})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message response", "error", err, "payload", string(data), "username", username)
		return
	}
	for _, ch := range a.pool {
		ch <- data
	}
	a.logger.InfoContext(ctx, "all notified")
}

func (a *App) OnMessage(ctx context.Context, payload []byte) error {
	username := getUsernameFromContext(ctx)
	a.logger.InfoContext(ctx, "received message", "connectionId", ctx.Value("connectionId").(string), "payload", string(payload), "username", username)

	var req MessageRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	data, err := json.Marshal(Message{Type: TypeChat, Data: MessageResponse{From: username, Message: req.Message}})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message response", "error", err, "payload", string(payload), "username", username)
		return fmt.Errorf("failed to marshal message response: %w", err)
	}

	switch req.To {
	case "":
		for _, ch := range a.pool {
			ch <- data
		}
	default:
		if ch, ok := a.pool[req.To]; ok {
			ch <- data
		}

		if ch, ok := a.pool[username]; ok {
			ch <- data
		}
	}
	return nil
}

func (a *App) MessageWriter(ctx context.Context, ch chan<- []byte) error {
	username := getUsernameFromContext(ctx)
	a.logger.InfoContext(ctx, "writing messages", "connectionId", ctx.Value("connectionId").(string), "username", username)
	if incoming, found := a.pool[username]; found {
		for {
			select {
			case msg, ok := <-incoming:
				if !ok {
					return nil
				}
				ch <- msg
			}
		}
	}

	return nil
}

func (a *App) UsernameExists(username string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.pool[username]
	return ok
}
