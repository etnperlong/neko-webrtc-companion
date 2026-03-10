package app

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/etnperlong/neko-webrtc-companion/internal/cloudflare"
	"github.com/etnperlong/neko-webrtc-companion/internal/config"
	"github.com/etnperlong/neko-webrtc-companion/internal/docker"
	httpserver "github.com/etnperlong/neko-webrtc-companion/internal/http"
	"github.com/etnperlong/neko-webrtc-companion/internal/refresh"
	"github.com/etnperlong/neko-webrtc-companion/internal/scheduler"
)

// App represents the turn refresh service runtime.
type App struct {
	cfg   config.Config
	ready atomic.Bool
	svc   *refresh.Service
}

// New constructs a new App with the provided configuration.
func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

// Run starts the service lifecycle and blocks until shutdown.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := a.initRuntime(); err != nil {
		return err
	}

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

	var opts []scheduler.Option
	if a.cfg.RunOnStart {
		opts = append(opts, scheduler.WithRunOnStart())
	}
	sched, err := scheduler.New(a.cfg.Cron, a.runScheduledJob, opts...)
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
	if a.svc == nil {
		return
	}
	_ = a.svc.RunOnce(ctx)
}

func (a *App) trigger(ctx context.Context) refresh.Result {
	if a.svc == nil {
		return refresh.Result{Err: errors.New("refresh service not configured")}
	}
	return a.svc.RunOnce(ctx)
}

func (a *App) initRuntime() error {
	fetcher, err := cloudflare.NewFetcher(cloudflare.FetcherConfig{
		KeyID:    a.cfg.CloudflareTURNKeyID,
		APIToken: a.cfg.CloudflareAPIToken,
		TTL:      a.cfg.CloudflareTURNTTL,
	})
	if err != nil {
		return fmt.Errorf("build cloudflare fetcher: %w", err)
	}

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return fmt.Errorf("build docker client: %w", err)
	}

	filters := docker.ContainerFilters{
		NamePattern:  a.cfg.DockerContainerNameGlob,
		ImagePattern: a.cfg.DockerImageGlob,
		LabelTrueKey: a.cfg.DockerLabelTrueKey,
	}
	restarter := docker.NewRestarter(dockerClient, filters)
	store := refresh.NewFileStore(a.cfg.NekoConfigPath, 0o600)
	rewriter := refresh.NewNekoRewriter()

	var restartTimeout *time.Duration
	if a.cfg.DockerRestartTimeout > 0 {
		timeout := a.cfg.DockerRestartTimeout
		restartTimeout = &timeout
	}

	a.svc = refresh.NewService(fetcher, rewriter, store, restarter, a.cfg.DockerContainerNameGlob, restartTimeout)
	return nil
}
