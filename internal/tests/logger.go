package tests

import (
	"context"
	"log/slog"
	"testing"
)

// TestHandler routes slog logs to a testing.T instance
type TestHandler struct {
	t testing.TB
}

func NewTestHandler(t testing.TB) *TestHandler {
	return &TestHandler{t: t}
}

func (h *TestHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // Capture all log levels during testing
}

func (h *TestHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format the message with its attributes
	// Note: For advanced setups, you can format this into JSON or use a standard text handler.
	h.t.Logf("[%s] %s", r.Level, r.Message)
	return nil
}

func (h *TestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Return handler as-is or implement to support context attributes
	return h
}

func (h *TestHandler) WithGroup(name string) slog.Handler {
	return h
}
