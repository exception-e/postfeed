package logger

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
)

type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if requestId, ok := ctx.Value(middleware.RequestIDKey).(string); ok && requestId != "" {
		r.AddAttrs(slog.String("request_id", requestId))
	}
	return h.Handler.Handle(ctx, r)
}

func (h ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h ContextHandler) WithGroup(name string) slog.Handler {
	return ContextHandler{Handler: h.Handler.WithGroup(name)}
}

func New(handler slog.Handler) *slog.Logger {
	return slog.New(ContextHandler{Handler: handler})
}
