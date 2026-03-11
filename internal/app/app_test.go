package app

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/etnperlong/neko-webrtc-companion/internal/refresh"
)

func TestLogRefreshResult_LogsFailureWithStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(original)

	logRefreshResult("scheduled", refresh.Result{Err: errors.New("boom")})

	output := buf.String()
	if !strings.Contains(output, "source=scheduled") {
		t.Fatalf("expected source field, got %q", output)
	}
	if !strings.Contains(output, "level=ERROR") {
		t.Fatalf("expected error level, got %q", output)
	}
	if !strings.Contains(output, "boom") {
		t.Fatalf("expected error details, got %q", output)
	}
}

func TestLogRefreshResult_LogsSuccessWithRestartCount(t *testing.T) {
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(original)

	logRefreshResult("scheduled", refresh.Result{Changed: true, RestartCount: 2, Restarted: []string{"room-a", "room-b"}})

	output := buf.String()
	if !strings.Contains(output, "restart_count=2") {
		t.Fatalf("expected restart count, got %q", output)
	}
	if !strings.Contains(output, "containers=room-a,room-b") {
		t.Fatalf("expected container list, got %q", output)
	}
}
