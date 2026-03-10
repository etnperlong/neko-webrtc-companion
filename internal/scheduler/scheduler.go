package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
)

// Option configures scheduler behavior.
type Option func(*Scheduler)

// WithRunOnStart executes the scheduled job immediately when the scheduler
// starts before the cron schedule is used.
func WithRunOnStart() Option {
	return func(s *Scheduler) {
		s.runOnStart = true
	}
}

// Scheduler represents a cron-driven job runner.
type Scheduler struct {
	cron       *cron.Cron
	job        func(context.Context)
	runOnStart bool

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// New constructs a scheduler that executes the provided job according to the cron
// spec.
func New(spec string, job func(context.Context), opts ...Option) (*Scheduler, error) {
	if job == nil {
		return nil, errors.New("job function is required")
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	c := cron.New(cron.WithParser(parser))

	s := &Scheduler{
		cron: c,
		job:  job,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	if _, err := s.cron.AddFunc(spec, s.invoke); err != nil {
		return nil, fmt.Errorf("schedule job: %w", err)
	}

	return s, nil
}

// Start begins the scheduler and hooks the job into the provided context scope.
// It returns an error if the scheduler has already been started.
func (s *Scheduler) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("scheduler already started")
	}

	jobCtx, cancel := context.WithCancel(ctx)
	s.ctx = jobCtx
	s.cancel = cancel
	s.started = true
	runOnStart := s.runOnStart
	s.mu.Unlock()

	if runOnStart {
		go s.invoke()
	}

	s.cron.Start()
	return nil
}

// Stop stops the scheduler and waits for any running jobs to complete.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.cancel = nil
	s.ctx = nil
	s.started = false
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := s.cron.Stop()
	<-done.Done()
	return nil
}

func (s *Scheduler) invoke() {
	ctx := s.jobContext()
	if ctx == nil {
		ctx = context.Background()
	}
	s.job(ctx)
}

func (s *Scheduler) jobContext() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}
