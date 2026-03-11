package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	envLogFormat = "LOG_FORMAT"
	envLogLevel  = "LOG_LEVEL"
	envLogColor  = "LOG_COLOR"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Config struct {
	Format Format
	Level  slog.Level
	Color  bool
}

func LoadConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := Config{Format: FormatText, Level: slog.LevelInfo, Color: true}

	if raw := strings.TrimSpace(getenv(envLogFormat)); raw != "" {
		format := Format(strings.ToLower(raw))
		switch format {
		case FormatText, FormatJSON:
			cfg.Format = format
		default:
			return Config{}, fmt.Errorf("parse %s: unsupported format %q", envLogFormat, raw)
		}
	}

	if raw := strings.TrimSpace(getenv(envLogLevel)); raw != "" {
		switch strings.ToLower(raw) {
		case "debug":
			cfg.Level = slog.LevelDebug
		case "info":
			cfg.Level = slog.LevelInfo
		case "warn", "warning":
			cfg.Level = slog.LevelWarn
		case "error":
			cfg.Level = slog.LevelError
		default:
			return Config{}, fmt.Errorf("parse %s: unsupported level %q", envLogLevel, raw)
		}
	}

	if raw := strings.TrimSpace(getenv(envLogColor)); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envLogColor, err)
		}
		cfg.Color = parsed
	}

	if cfg.Format == FormatJSON {
		cfg.Color = false
	}

	return cfg, nil
}
