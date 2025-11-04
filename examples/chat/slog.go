package main

import (
	"context"
	"log/slog"
	"os"
)

func Slog() *slog.Logger {
	return slog.New(UsernameSlogHandler(slog.NewJSONHandler(os.Stdout, nil)))
}

func UsernameSlogHandler(handler slog.Handler) slog.Handler {
	return &usernameSlogHandler{Handler: handler}
}

type usernameSlogHandler struct {
	slog.Handler
}

func (h *usernameSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if val := getUsernameFromContext(ctx); val != "" {
		r.AddAttrs(slog.Any("username", val))
	}

	return h.Handler.Handle(ctx, r)
}
