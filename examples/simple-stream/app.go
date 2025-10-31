package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

func NewApp(logger *slog.Logger) *App {
	return &App{
		logger: logger,
		pool:   &sync.Map{},
	}
}

type App struct {
	logger *slog.Logger
	pool   *sync.Map
}

func (a *App) OnConnect(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	a.pool.Store(username, NewClient(ctx))
	go a.provideActiveClients(ctx)
	go a.notifyAllForConnection(ctx)
}

func (a *App) OnDisconnect(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	if client := a.getClient(username); client != nil {
		a.pool.Delete(username)
		client.Close()
	}

	go a.notifyAllForDisconnection(ctx)
}

func (a *App) OnMessage(ctx context.Context, payload []byte) error {
	username := getUsernameFromContext(ctx)
	a.logger.InfoContext(ctx, "received message", "payload", string(payload))

	var req MessageRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	data, err := json.Marshal(Message{Type: TypeChat, Data: MessageResponse{From: username, Message: req.Message}})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message", "error", err, "payload", string(payload))
		return err
	}

	switch req.To {
	case "":
		a.notifyAll(data)
	default:
		if client := a.getClient(req.To); client != nil {
			client.Send(data)
		}

		if client := a.getClient(username); client != nil {
			client.Send(data)
		}
	}
	return nil
}

func (a *App) MessageWriter(ctx context.Context) (<-chan []byte, error) {
	a.logger.InfoContext(ctx, "writing messages")

	username := getUsernameFromContext(ctx)
	client := a.getClient(username)
	if client == nil {
		a.logger.ErrorContext(ctx, "channel not found for username", "username", username)
		return nil, fmt.Errorf("channel not found for username %s", username)
	}

	return client.Channel(), nil
}

func (a *App) UsernameExists(username string) bool {
	_, ok := a.pool.Load(username)
	return ok
}

func (a *App) getClient(username string) *Client {
	client, ok := a.pool.Load(username)
	if !ok {
		return nil
	}

	return client.(*Client)
}

func (a *App) notifyAllForConnection(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	payload, err := json.Marshal(Message{Type: TypeUserConnected, Data: username})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message", "error", err, "payload", string(payload))
		return
	}

	a.notifyAll(payload, username)
}

func (a *App) notifyAllForDisconnection(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	payload, err := json.Marshal(Message{Type: TypeUserDisconnected, Data: username})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message", "error", err, "payload", string(payload))
		return
	}

	a.notifyAll(payload, username)
}

func (a *App) notifyAll(payload []byte, filtered ...string) {
	a.pool.Range(func(key, value interface{}) bool {
		if slices.Contains(filtered, key.(string)) {
			return true
		}

		client := value.(*Client)
		client.Send(payload)
		return true
	})
}

func (a *App) provideActiveClients(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	activeClients := a.allActiveClients(username)
	if len(activeClients) == 0 {
		return
	}

	payload, err := json.Marshal(Message{Type: TypeClientList, Data: activeClients})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message", "error", err, "payload", string(payload))
		return
	}

	if client := a.getClient(username); client != nil {
		client.Send(payload)
	}
}

func (a *App) allActiveClients(filtered ...string) []string {
	var clients []string
	a.pool.Range(func(key, value interface{}) bool {
		if slices.Contains(filtered, key.(string)) {
			return true
		}

		clients = append(clients, key.(string))
		return true
	})

	return clients
}
