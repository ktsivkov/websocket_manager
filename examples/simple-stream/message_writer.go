package main

import (
	"context"
	"log/slog"

	"github.com/ktsivkov/websocket_manager"
)

func getMessageWriter(logger *slog.Logger, pool *ConnectionPool) websocket_manager.MessageWriterFunc {
	return func(ctx context.Context, ch chan<- []byte) error {
		connId := ctx.Value("connectionId").(string)
		msgStream := pool.Register(connId)
		defer pool.Unregister(connId)

		for {
			select {
			case <-ctx.Done():
				return nil
			case msg := <-msgStream:
				ch <- msg
				logger.InfoContext(ctx, "sent message", "payload", string(msg), "connectionId", connId)
			}
		}
	}
}
