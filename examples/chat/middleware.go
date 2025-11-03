package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

var connectionIdMiddleware websocket_manager.MiddlewareFunc = func(conn *websocket_manager.Connection) (*websocket_manager.Connection, error) {
	return websocket_manager.ConnectionWithContext(context.WithValue(conn.Context(), "connectionId", time.Now().String()), conn), nil
}

var errUsernameTaken = errors.New("username taken")

func getUsernameMiddleware(app *App) websocket_manager.MiddlewareFunc {
	return func(conn *websocket_manager.Connection) (*websocket_manager.Connection, error) {
		username := conn.Context().Value("username").(string)
		if app.UsernameExists(username) {
			app.logger.InfoContext(conn.Context(), "username taken", "username", username)

			data, err := json.Marshal(ControlMessage{
				Code:    "USERNAME_ALREADY_TAKEN",
				Message: "This username is already taken. Please choose another one.",
			})
			if err != nil {
				app.logger.ErrorContext(conn.Context(), "failed to marshal close message", "error", err)
				return nil, fmt.Errorf("%w: %w", err, errUsernameTaken)
			}

			if err := websocket_manager.CloseMessage(websocket.ClosePolicyViolation, string(data), 5*time.Second).Write(conn); err != nil {
				app.logger.ErrorContext(conn.Context(), "failed to write close message", "error", err)
				return nil, fmt.Errorf("%w: %w", err, errUsernameTaken)
			}

			return nil, errUsernameTaken
		}

		return conn, nil
	}
}
