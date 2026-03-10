package main

import (
	"log"
	"os"

	"github.com/etnperlong/neko-webrtc-companion/internal/app"
	"github.com/etnperlong/neko-webrtc-companion/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv(os.Getenv)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	runner := app.New(cfg)
	if err := runner.Run(); err != nil {
		log.Fatalf("application stopped: %v", err)
	}
}
