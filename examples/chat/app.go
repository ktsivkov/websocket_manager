package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

func NewApp(logger *slog.Logger) *App {
	return &App{
		logger: logger,
		pool:   &sync.Map{},
		wg:     &sync.WaitGroup{},
	}
}

type App struct {
	logger *slog.Logger
	pool   *sync.Map
	wg     *sync.WaitGroup
}

func (a *App) OnConnect(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	a.logger.InfoContext(ctx, "client connected", "username", username)
	a.pool.Store(username, NewClient(ctx))
	go a.provideActiveClients(ctx)
	go a.notifyAllForConnection(ctx)
	a.wg.Add(1)
}

func (a *App) OnDisconnect(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	a.logger.InfoContext(ctx, "client disconnected", "username", username)
	if client := a.getClient(username); client != nil {
		a.pool.Delete(username)
		client.Close()
	}

	go a.notifyAllForDisconnection(ctx)
	a.wg.Done()
}

func (a *App) OnMessage(ctx context.Context, payload []byte) {
	username := getUsernameFromContext(ctx)

	sourceClient := a.getClient(username)
	if sourceClient == nil {
		sourceClient.SendMessage(websocket_manager.CloseMessage(websocket.CloseInternalServerErr, "Internal server error.", 5*time.Second))
		a.logger.ErrorContext(ctx, "client not found for username", "username", username)
		return
	}

	var req MessageRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		a.logger.ErrorContext(ctx, "failed to unmarshal message", "error", err, "payload", string(payload))
		sourceClient.SendMessage(websocket_manager.CloseMessage(websocket.ClosePolicyViolation, "Bad message format.", 5*time.Second))
		return
	}

	if req.Message == "/leave" {
		a.leave(sourceClient)
		return
	}

	data, err := json.Marshal(Message{Type: TypeChat, Data: MessageResponse{From: username, Message: req.Message}})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to marshal message", "error", err, "payload", string(payload))
		sourceClient.SendMessage(websocket_manager.CloseMessage(websocket.CloseInternalServerErr, "Internal server error.", 5*time.Second))
		return
	}

	msg, err := websocket_manager.TextMessage(string(data))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create message", "error", err, "payload", string(payload))
		sourceClient.SendMessage(websocket_manager.CloseMessage(websocket.CloseInternalServerErr, "Internal server error.", 5*time.Second))
		return
	}

	switch req.To {
	case "":
		a.notifyAll(msg)
	default:
		if client := a.getClient(req.To); client != nil {
			client.SendMessage(msg)
		}

		sourceClient.SendMessage(msg)
	}
}

func (a *App) MessageWriter(ctx context.Context) (<-chan websocket_manager.Message, error) {
	a.logger.InfoContext(ctx, "writing messages")

	username := getUsernameFromContext(ctx)
	client := a.getClient(username)
	if client == nil {
		a.logger.ErrorContext(ctx, "client not found for username", "username", username)
		return nil, fmt.Errorf("client not found for username %s", username)
	}

	return client.WriteChannel(), nil
}

func (a *App) Shutdown(ctx context.Context) error {
	ch := make(chan struct{}, 1)
	go func() {
		a.pool.Range(func(key, value interface{}) bool {
			a.pool.Delete(key)
			client := value.(*Client)
			client.SendMessage(websocket_manager.CloseMessage(websocket.CloseServiceRestart, "Goodbye.", 5*time.Second))
			client.Close()
			return true
		})
		a.wg.Wait()
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
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
	payload, err := a.createWsMessage(Message{Type: TypeUserConnected, Data: username})
	if err != nil {
		a.logger.ErrorContext(ctx, "could not create websocket message")
		return
	}

	a.notifyAll(payload, username)
}

func (a *App) notifyAllForDisconnection(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	payload, err := a.createWsMessage(Message{Type: TypeUserDisconnected, Data: username})
	if err != nil {
		a.logger.ErrorContext(ctx, "could not create websocket message")
		return
	}

	a.notifyAll(payload, username)
}

func (a *App) notifyAll(msg websocket_manager.Message, filtered ...string) {
	a.pool.Range(func(key, value interface{}) bool {
		if slices.Contains(filtered, key.(string)) {
			return true
		}

		client := value.(*Client)
		client.SendMessage(msg)
		return true
	})
}

func (a *App) provideActiveClients(ctx context.Context) {
	username := getUsernameFromContext(ctx)
	activeClients := a.allActiveClients(username)
	if len(activeClients) == 0 {
		return
	}

	payload, err := a.createWsMessage(Message{Type: TypeClientList, Data: activeClients})
	if err != nil {
		a.logger.ErrorContext(ctx, "could not create websocket message")
		return
	}

	if client := a.getClient(username); client != nil {
		client.SendMessage(payload)
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

func (a *App) createWsMessage(msg Message) (websocket_manager.Message, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return websocket_manager.TextMessage(string(payload))
}

func (a *App) leave(client *Client) {
	client.SendMessage(websocket_manager.CloseMessage(websocket.CloseNormalClosure, "Goodbye.", 5*time.Second))
}
