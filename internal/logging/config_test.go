package logging

import (
	"log/slog"
	"testing"
)

func TestLoadConfig_DefaultsToColoredTextInfo(t *testing.T) {
	cfg, err := LoadConfig(func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Format != FormatText {
		t.Fatalf("expected text format, got %q", cfg.Format)
	}
	if cfg.Level != slog.LevelInfo {
		t.Fatalf("expected info level, got %v", cfg.Level)
	}
	if !cfg.Color {
		t.Fatal("expected color enabled by default")
	}
}

func TestLoadConfig_RejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "format", env: map[string]string{"LOG_FORMAT": "xml"}},
		{name: "level", env: map[string]string{"LOG_LEVEL": "trace"}},
		{name: "color", env: map[string]string{"LOG_COLOR": "maybe"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConfig(func(key string) string { return tc.env[key] })
			if err == nil {
				t.Fatal("expected config error")
			}
		})
	}
}
