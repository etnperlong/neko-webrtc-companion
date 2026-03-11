package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/etnperlong/neko-webrtc-companion/internal/app"
	"github.com/etnperlong/neko-webrtc-companion/internal/config"
	"github.com/etnperlong/neko-webrtc-companion/internal/logging"
)

func main() {
	os.Exit(run(os.Getenv, os.Stderr))
}

func run(getenv func(string) string, stderr io.Writer) int {
	logCfg, err := logging.LoadConfig(getenv)
	if err != nil {
		logging.New(logging.Config{Format: logging.FormatText, Level: slog.LevelInfo, Color: false}, stderr).Error("failed to load logging config", "component", "main", "err", err)
		return 1
	}

	logger := logging.New(logCfg, stderr)
	slog.SetDefault(logger)

	cfg, err := config.LoadFromEnv(getenv)
	if err != nil {
		slog.Error("failed to load config", "component", "main", "err", err)
		return 1
	}

	slog.Info("starting neko turn refresh", "component", "main", "http_addr", cfg.HTTPAddr, "cron", cfg.Cron, "config_path", cfg.NekoConfigPath, "run_on_start", cfg.RunOnStart)

	runner := app.New(cfg)
	if err := runner.Run(); err != nil {
		slog.Error("application stopped", "component", "main", "err", err)
		return 1
	}

	return 0
}
