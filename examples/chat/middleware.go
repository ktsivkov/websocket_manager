package main

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ktsivkov/websocket_manager"
)

var connectionIdMiddleware websocket_manager.MiddlewareFunc = func(conn *websocket_manager.Connection) (*websocket_manager.Connection, error) {
	return websocket_manager.ConnectionWithContext(context.WithValue(conn.Context(), "connectionId", time.Now().String()), conn), nil
}

func getUsernameMiddleware(app *App) websocket_manager.MiddlewareFunc {
	return func(conn *websocket_manager.Connection) (*websocket_manager.Connection, error) {
		username := conn.Context().Value("username").(string)
		if app.UsernameExists(username) {
			_ = conn.WriteControlClose(websocket.ClosePolicyViolation, websocket_manager.ControlMessage{
				Code:    "USERNAME_ALREADY_TAKEN",
				Message: "This username is already taken. Please choose another one.",
			}, 5*time.Second)
			return nil, errors.New("username is taken")
		}

		return conn, nil
	}
}
