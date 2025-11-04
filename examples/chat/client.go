package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

func NewClient(ctx context.Context, logger *slog.Logger, username string, bus *Bus) *Client {
	return &Client{
		ctx:          ctx,
		username:     username,
		logger:       logger,
		bus:          bus,
		writeChannel: make(chan websocket_manager.Message),
	}
}

type Client struct {
	ctx          context.Context
	username     string
	logger       *slog.Logger
	bus          *Bus
	writeChannel chan websocket_manager.Message
}

func (c *Client) OnConnect() {
	if err := c.bus.Subscribe(c); err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			msg, err := createWsCloseMessage(ControlMessageUsernameTaken)
			if err != nil {
				c.logger.ErrorContext(c.ctx, "failed to create message", "error", err, "data", ControlMessageUsernameTaken)
				return
			}

			c.SendMessage(msg)
			return
		}

		c.logger.ErrorContext(c.ctx, "failed to subscribe", "error", err)
		c.SendMessage(websocket_manager.CloseMessage(websocket.ClosePolicyViolation, "Failed to join.", 5*time.Second))
		return
	}

	c.logger.InfoContext(c.ctx, "client connected")

	go c.sendListOfActiveClients()
	go c.notifyAllForConnection()
}

func (c *Client) OnDisconnect(msg *websocket_manager.ClientCloseMessage) {
	if msg != nil {
		c.logger.InfoContext(c.ctx, "closing websocket connection upon a client request", "code", msg.Code, "text", msg.Text)
	}

	defer close(c.writeChannel)

	if err := c.bus.Unsubscribe(c); err != nil {
		c.logger.ErrorContext(c.ctx, "failed to unsubscribe", "error", err)
	}

	c.logger.InfoContext(c.ctx, "client disconnected")

	go c.notifyAllForDisconnection()
}

func (c *Client) OnMessage(payload []byte) {
	c.logger.InfoContext(c.ctx, "received message", "payload", string(payload))

	var req MessageRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		c.logger.ErrorContext(c.ctx, "failed to unmarshal message", "error", err, "payload", string(payload))
		c.SendMessage(websocket_manager.CloseMessage(websocket.ClosePolicyViolation, "Bad message format.", 5*time.Second))
		return
	}

	if req.Message == "/leave" {
		c.Kick()
		return
	}

	data, err := json.Marshal(Message{Type: TypeChat, Data: MessageResponse{From: c.Username(), Message: req.Message}})
	if err != nil {
		c.logger.ErrorContext(c.ctx, "failed to marshal message", "error", err, "payload", string(payload))
		c.SendMessage(websocket_manager.CloseMessage(websocket.CloseInternalServerErr, "Internal server error.", 5*time.Second))
		return
	}

	msg, err := websocket_manager.TextMessage(string(data))
	if err != nil {
		c.logger.ErrorContext(c.ctx, "failed to create message", "error", err, "payload", string(payload))
		c.SendMessage(websocket_manager.CloseMessage(websocket.CloseInternalServerErr, "Internal server error.", 5*time.Second))
		return
	}

	switch req.To {
	case "":
		c.bus.NotifyAll(msg)
	default:
		err := c.bus.Notify(req.To, msg)
		if err != nil {
			c.logger.ErrorContext(c.ctx, "failed to notify", "error", err, "payload", string(payload))
			return
		}

		c.SendMessage(msg)
	}
}

func (c *Client) WriterChannel() <-chan websocket_manager.Message {
	return c.writeChannel
}

func (c *Client) SendMessage(msg websocket_manager.Message) {
	c.writeChannel <- msg
}

func (c *Client) Kick() {
	c.SendMessage(websocket_manager.CloseMessage(websocket.CloseNormalClosure, "Goodbye.", 5*time.Second))
}

func (c *Client) NotifyShutdown() {
	c.SendMessage(websocket_manager.CloseMessage(websocket.CloseServiceRestart, "Service is about to restart.", 5*time.Second))
}

func (c *Client) Username() string {
	return c.username
}

func (c *Client) sendListOfActiveClients() {
	activeClients := c.bus.Active(c.Username())
	if len(activeClients) == 0 {
		return
	}

	payload, err := createWsMessage(Message{Type: TypeClientList, Data: activeClients})
	if err != nil {
		c.logger.ErrorContext(c.ctx, "could not create websocket message")
		return
	}
	c.SendMessage(payload)
}

func (c *Client) notifyAllForConnection() {
	payload, err := createWsMessage(Message{Type: TypeUserConnected, Data: c.Username()})
	if err != nil {
		c.logger.ErrorContext(c.ctx, "could not create websocket message")
		return
	}

	c.bus.NotifyAll(payload, c.Username())
}

func (c *Client) notifyAllForDisconnection() {
	payload, err := createWsMessage(Message{Type: TypeUserDisconnected, Data: c.Username()})
	if err != nil {
		c.logger.ErrorContext(c.ctx, "could not create websocket message")
		return
	}

	c.bus.NotifyAll(payload, c.Username())
}

func createWsMessage(msg Message) (websocket_manager.Message, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return websocket_manager.TextMessage(string(payload))
}

func createWsCloseMessage(msg ControlMessage) (websocket_manager.Message, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return websocket_manager.CloseMessage(websocket.ClosePolicyViolation, string(payload), 5*time.Second), nil
}
