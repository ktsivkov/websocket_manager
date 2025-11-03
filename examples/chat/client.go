package main

import (
	"context"

	"github.com/ktsivkov/websocket_manager"
)

func NewClient(ctx context.Context) *Client {
	return &Client{
		ctx:          ctx,
		writeChannel: make(chan websocket_manager.Message),
	}
}

type Client struct {
	ctx          context.Context
	writeChannel chan websocket_manager.Message
}

func (c *Client) SendMessage(msg websocket_manager.Message) {
	select {
	case <-c.ctx.Done():
		return
	case c.writeChannel <- msg:
	}
}

func (c *Client) WriteChannel() <-chan websocket_manager.Message {
	return c.writeChannel
}

func (c *Client) Close() {
	close(c.writeChannel)
}
