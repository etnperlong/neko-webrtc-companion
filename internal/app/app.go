package app

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/etnperlong/neko-webrtc-companion/internal/config"
	httpserver "github.com/etnperlong/neko-webrtc-companion/internal/http"
	"github.com/etnperlong/neko-webrtc-companion/internal/scheduler"
)

// App represents the turn refresh service runtime.
type App struct {
	cfg   config.Config
	ready atomic.Bool
}

// New constructs a new App with the provided configuration.
func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

// Run starts the service lifecycle and blocks until shutdown.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	handler := httpserver.New(httpserver.Dependencies{
		Ready:   func() bool { return a.ready.Load() },
		Trigger: a.trigger,
	})
	httpServer := &nethttp.Server{
		Addr:    a.cfg.HTTPAddr,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			_ = httpServer.Shutdown(context.Background())
			return fmt.Errorf("serve http: %w", err)
		}
	default:
	}

	sched, err := scheduler.New(a.cfg.Cron, a.runScheduledJob, scheduler.WithRunOnStart())
	if err != nil {
		_ = httpServer.Shutdown(context.Background())
		return fmt.Errorf("build scheduler: %w", err)
	}

	if err := sched.Start(ctx); err != nil {
		_ = httpServer.Shutdown(context.Background())
		return fmt.Errorf("start scheduler: %w", err)
	}

	a.ready.Store(true)
	defer a.ready.Store(false)

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			runErr = fmt.Errorf("serve http: %w", err)
		}
	}

	if err := httpServer.Shutdown(context.Background()); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		if runErr == nil {
			runErr = fmt.Errorf("shutdown http server: %w", err)
		} else {
			runErr = fmt.Errorf("%v; shutdown http server: %w", runErr, err)
		}
	}

	if err := sched.Stop(); err != nil {
		if runErr == nil {
			runErr = fmt.Errorf("stop scheduler: %w", err)
		} else {
			runErr = fmt.Errorf("%v; stop scheduler: %w", runErr, err)
		}
	}

	return runErr
}

func (a *App) runScheduledJob(ctx context.Context) {
	_ = ctx
	// TODO: wire refresh service job once dependencies are available.
}

func (a *App) trigger(ctx context.Context) error {
	a.runScheduledJob(ctx)
	return nil
}
