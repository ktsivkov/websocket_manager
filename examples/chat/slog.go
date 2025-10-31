package main

import (
	"context"
	"log/slog"
	"os"
)

func Slog() *slog.Logger {
	return slog.New(ConnectionIdSlogHandler(UsernameSlogHandler(slog.NewJSONHandler(os.Stdout, nil))))
}

func ConnectionIdSlogHandler(handler slog.Handler) slog.Handler {
	return &connectionIdSlogHandler{Handler: handler}
}

type connectionIdSlogHandler struct {
	slog.Handler
}

func (h *connectionIdSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if val := getConnectionIdFromContext(ctx); val != "" {
		r.AddAttrs(slog.Any("connectionId", val))
	}

	return h.Handler.Handle(ctx, r)
}

func UsernameSlogHandler(handler slog.Handler) slog.Handler {
	return &usernameSlogHandler{Handler: handler}
}

type usernameSlogHandler struct {
	slog.Handler
}

func (h *usernameSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if val := getUsernameFromContext(ctx); val != "" {
		r.AddAttrs(slog.Any("connectionId", val))
	}

	return h.Handler.Handle(ctx, r)
}
