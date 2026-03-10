package app

import "github.com/etnperlong/neko-webrtc-companion/internal/config"

// App represents the turn refresh service runtime.
type App struct {
	cfg config.Config
}

// New constructs a new App with the provided configuration.
func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

// Run starts the service.
func (a *App) Run() error {
	_ = a.cfg
	return nil
}
