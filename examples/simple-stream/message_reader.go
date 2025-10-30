package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ktsivkov/websocket_manager"
)

func getMessageReader(logger *slog.Logger, pool *ConnectionPool) websocket_manager.MessageReaderFunc {
	return func(ctx context.Context, payload []byte) error {
		logger.InfoContext(ctx, "received message", "connectionId", ctx.Value("connectionId").(string), "payload", string(payload))
		username := ctx.Value("username").(string)
		var req MessageRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return err
		}

		data, err := json.Marshal(MessageResponse{From: username, Message: req.Message})
		if err != nil {
			logger.ErrorContext(ctx, "failed to marshal message response", "error", err)
			return fmt.Errorf("failed to marshal message response: %w", err)
		}

		switch req.To {
		case "":
			logger.InfoContext(ctx, "sending message to all", "connectionId", ctx.Value("connectionId").(string), "payload", req.Message)
			pool.SendToAll(data)
		default:

			logger.InfoContext(ctx, "sending message", "connectionId", ctx.Value("connectionId").(string), "to", req.To, "payload", string(data))
			pool.SendTo(req.To, data)
		}
		return nil
	}
}
