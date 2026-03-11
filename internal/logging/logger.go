package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

const ansiReset = "\x1b[0m"

var levelColors = map[slog.Level]string{
	slog.LevelDebug: "\x1b[36m",
	slog.LevelInfo:  "\x1b[32m",
	slog.LevelWarn:  "\x1b[33m",
	slog.LevelError: "\x1b[31m",
}

func New(cfg Config, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}

	if cfg.Format == FormatJSON {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: cfg.Level}))
	}

	opts := &slog.HandlerOptions{Level: cfg.Level}
	if cfg.Color {
		opts.ReplaceAttr = func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.LevelKey {
				return attr
			}
			if level, ok := attr.Value.Any().(slog.Level); ok {
				return slog.String(attr.Key, colorizeLevel(level))
			}
			return attr
		}
	}

	return slog.New(&syncHandler{Handler: slog.NewTextHandler(w, opts)})
}

func colorizeLevel(level slog.Level) string {
	label := level.String()
	if color, ok := levelColors[level]; ok {
		return color + label + ansiReset
	}
	return label
}

type syncHandler struct {
	slog.Handler
	mu sync.Mutex
}

func (h *syncHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Handler.Handle(ctx, record)
}

func (h *syncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &syncHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *syncHandler) WithGroup(name string) slog.Handler {
	return &syncHandler{Handler: h.Handler.WithGroup(name)}
}
