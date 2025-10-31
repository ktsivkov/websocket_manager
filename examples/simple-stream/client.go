package main

import (
	"context"
)

func NewClient(ctx context.Context) *Client {
	return &Client{
		ctx:     ctx,
		channel: make(chan []byte),
	}
}

type Client struct {
	ctx     context.Context
	channel chan []byte
}

func (c *Client) Send(payload []byte) {
	select {
	case <-c.ctx.Done():
		return
	case c.channel <- payload:
	}
}

func (c *Client) Channel() <-chan []byte {
	return c.channel
}

func (c *Client) Close() {
	close(c.channel)
}
