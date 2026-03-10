package refresh

import (
	"context"
	"sync"
	"time"

	"github.com/etnperlong/neko-webrtc-companion/internal/cloudflare"
	"github.com/etnperlong/neko-webrtc-companion/internal/neko"
)

// TURNClient represents the dependency that provides TURN servers.
type TURNClient interface {
	Fetch(ctx context.Context) ([]cloudflare.ICEServer, error)
}

// Rewriter handles rewriting the YAML document with the requested ICE servers.
type Rewriter interface {
	Rewrite(ctx context.Context, current []byte, servers []neko.ICEServer) ([]byte, bool, error)
}

// Store reads and writes the YAML configuration file.
type Store interface {
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, data []byte) error
}

// Restarter restarts containers whose names match a configured glob.
type Restarter interface {
	RestartMatching(ctx context.Context, pattern string, timeout *time.Duration) ([]string, error)
}

// Service orchestrates a single refresh job.
type Service struct {
	fetcher        TURNClient
	rewriter       Rewriter
	store          Store
	restarter      Restarter
	containerGlob  string
	restartTimeout *time.Duration

	mu      sync.Mutex
	running bool
}

// Result captures the outcome of a refresh job.
type Result struct {
	Err          error
	Busy         bool
	Skipped      bool
	Changed      bool
	NoOp         bool
	RestartCount int
	Restarted    []string
}

// NewService creates a Service with the provided dependencies.
func NewService(fetcher TURNClient, rewriter Rewriter, store Store, restarter Restarter, containerGlob string, restartTimeout *time.Duration) *Service {
	return &Service{
		fetcher:        fetcher,
		rewriter:       rewriter,
		store:          store,
		restarter:      restarter,
		containerGlob:  containerGlob,
		restartTimeout: restartTimeout,
	}
}

// RunOnce executes one refresh job.
func (s *Service) RunOnce(ctx context.Context) Result {
	result := Result{}
	if !s.tryStart() {
		result.Busy = true
		result.Skipped = true
		return result
	}
	defer s.finish()

	servers, err := s.fetcher.Fetch(ctx)
	if err != nil {
		result.Err = err
		return result
	}

	current, err := s.store.Read(ctx)
	if err != nil {
		result.Err = err
		return result
	}

	rewritten, changed, err := s.rewriter.Rewrite(ctx, current, toNekoICEServers(servers))
	if err != nil {
		result.Err = err
		return result
	}

	result.Changed = changed
	result.NoOp = !changed
	if !changed {
		return result
	}

	if err := s.store.Write(ctx, rewritten); err != nil {
		result.Err = err
		return result
	}

	restarted, err := s.restarter.RestartMatching(ctx, s.containerGlob, s.restartTimeout)
	if err != nil {
		result.Err = err
		return result
	}

	result.Restarted = restarted
	result.RestartCount = len(restarted)

	return result
}

func (s *Service) tryStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Service) finish() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

func toNekoICEServers(servers []cloudflare.ICEServer) []neko.ICEServer {
	result := make([]neko.ICEServer, len(servers))
	for i, server := range servers {
		result[i] = neko.ICEServer{
			URLs:       append([]string(nil), server.URLs...),
			Username:   server.Username,
			Credential: server.Credential,
		}
	}
	return result
}
