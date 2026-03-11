package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_JSONDisablesColorAndEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatJSON, Level: slog.LevelInfo, Color: true}, &buf)
	logger.Info("hello", "component", "test")

	output := buf.String()
	if !strings.Contains(output, `"msg":"hello"`) {
		t.Fatalf("expected json output, got %q", output)
	}
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("expected json output without ANSI color, got %q", output)
	}
}

func TestNew_TextCanColorizeLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatText, Level: slog.LevelInfo, Color: true}, &buf)
	logger.Error("boom", "component", "test")

	output := buf.String()
	if !strings.Contains(output, `\x1b[31mERROR\x1b[0m`) {
		t.Fatalf("expected ANSI color in text output, got %q", output)
	}
	if !strings.Contains(output, "boom") {
		t.Fatalf("expected message in output, got %q", output)
	}
}
