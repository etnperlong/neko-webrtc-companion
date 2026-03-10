package refresh

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/etnperlong/neko-webrtc-companion/internal/cloudflare"
	"github.com/etnperlong/neko-webrtc-companion/internal/neko"
)

func TestRunOnce_CloudflareFailureSkipsWriteAndRestart(t *testing.T) {
	fetchErr := errors.New("cloudflare failure")
	fetcher := &fakeFetcher{err: fetchErr}
	service := NewService(fetcher, &fakeRewriter{}, &fakeStore{}, &fakeRestarter{}, "", nil)

	result := service.RunOnce(context.Background())

	if !errors.Is(result.Err, fetchErr) {
		t.Fatalf("expected error %v, got %v", fetchErr, result.Err)
	}

	if result.RestartCount != 0 {
		t.Fatalf("expected zero restarts, got %d", result.RestartCount)
	}
}

func TestRunOnce_RewriteFailureSkipsWriteAndRestart(t *testing.T) {
	rewriteErr := errors.New("rewrite failure")
	recorder := &callRecorder{}
	store := &fakeStore{recorder: recorder}
	restarter := &fakeRestarter{recorder: recorder}
	service := NewService(&fakeFetcher{recorder: recorder}, &fakeRewriter{err: rewriteErr, recorder: recorder}, store, restarter, "", nil)

	result := service.RunOnce(context.Background())

	if !errors.Is(result.Err, rewriteErr) {
		t.Fatalf("expected rewrite error, got %v", result.Err)
	}
	if len(store.writes) != 0 {
		t.Fatalf("write should not occur on rewrite failure, got %d", len(store.writes))
	}
	if restarter.called {
		t.Fatalf("restart should not occur on rewrite failure")
	}
}

func TestRunOnce_WriteFailureSkipsRestart(t *testing.T) {
	writeErr := errors.New("write failure")
	recorder := &callRecorder{}
	store := &fakeStore{writeErr: writeErr, recorder: recorder}
	restarter := &fakeRestarter{recorder: recorder}
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true, recorder: recorder}
	service := NewService(&fakeFetcher{recorder: recorder}, rewriter, store, restarter, "", nil)

	result := service.RunOnce(context.Background())

	if !errors.Is(result.Err, writeErr) {
		t.Fatalf("expected write error, got %v", result.Err)
	}
	if restarter.called {
		t.Fatalf("restart should not occur when write fails")
	}
	if len(store.writes) != 1 {
		t.Fatalf("expected one write attempt, got %d", len(store.writes))
	}
}

func TestRunOnce_NoOpSkipsWriteAndRestart(t *testing.T) {
	recorder := &callRecorder{}
	store := &fakeStore{recorder: recorder}
	restarter := &fakeRestarter{recorder: recorder}
	rewriter := &fakeRewriter{result: []byte("unchanged"), changed: false, recorder: recorder}
	service := NewService(&fakeFetcher{recorder: recorder}, rewriter, store, restarter, "", nil)

	result := service.RunOnce(context.Background())

	if result.Err != nil {
		t.Fatalf("expected no error for no-op, got %v", result.Err)
	}
	if result.Changed {
		t.Fatalf("expected no change flag on no-op result")
	}
	if !result.NoOp {
		t.Fatalf("expected NoOp true when nothing changed")
	}
	if len(store.writes) != 0 {
		t.Fatalf("write should not occur on no-op")
	}
	if restarter.called {
		t.Fatalf("restart should not occur on no-op")
	}
	if result.RestartCount != 0 {
		t.Fatalf("expected zero restarts for no-op, got %d", result.RestartCount)
	}
}

func TestRunOnce_SuccessWritesBeforeRestartAndRecordsResult(t *testing.T) {
	recorder := &callRecorder{}
	servers := []cloudflare.ICEServer{{URLs: []string{"turn.example.com"}, Username: "u", Credential: "p"}}
	fetcher := &fakeFetcher{servers: servers, recorder: recorder}
	store := &fakeStore{readData: []byte("existing"), recorder: recorder}
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true, recorder: recorder}
	restartedNames := []string{"neko-turn-1", "neko-turn-2"}
	restarter := &fakeRestarter{restarted: restartedNames, recorder: recorder}
	timeout := 5 * time.Second
	service := NewService(fetcher, rewriter, store, restarter, "neko-*", &timeout)

	result := service.RunOnce(context.Background())

	if result.Err != nil {
		t.Fatalf("expected success, got error %v", result.Err)
	}
	if !result.Changed {
		t.Fatalf("expected change to be reported")
	}
	if result.NoOp {
		t.Fatalf("expected NoOp false on change")
	}
	if result.RestartCount != len(restartedNames) {
		t.Fatalf("expected restart count %d, got %d", len(restartedNames), result.RestartCount)
	}
	if len(result.Restarted) != len(restartedNames) {
		t.Fatalf("expected restarted names %v, got %v", restartedNames, result.Restarted)
	}
	if len(store.writes) != 1 {
		t.Fatalf("expected a single write, got %d", len(store.writes))
	}
	expectedOrder := []string{"fetch", "read", "rewrite", "write", "restart"}
	if len(recorder.ops) != len(expectedOrder) {
		t.Fatalf("expected %d operations, got %d", len(expectedOrder), len(recorder.ops))
	}
	for i, op := range expectedOrder {
		if recorder.ops[i] != op {
			t.Fatalf("expected operation %q at position %d, got %q", op, i, recorder.ops[i])
		}
	}
	if restarter.pattern != "neko-*" {
		t.Fatalf("expected restart pattern to be forwarded, got %q", restarter.pattern)
	}
	if restarter.timeout == nil || *restarter.timeout != timeout {
		t.Fatalf("expected restart timeout %v, got %v", timeout, restarter.timeout)
	}
}

func TestRunOnce_ReadFailureSkipsRewriteWriteRestart(t *testing.T) {
	readErr := errors.New("read failure")
	rewriter := &fakeRewriter{}
	store := &fakeStore{readErr: readErr}
	service := NewService(&fakeFetcher{}, rewriter, store, &fakeRestarter{}, "", nil)

	result := service.RunOnce(context.Background())

	if !errors.Is(result.Err, readErr) {
		t.Fatalf("expected read error, got %v", result.Err)
	}
	if rewriter.called {
		t.Fatalf("rewriter should not be invoked when read fails")
	}
	if len(store.writes) != 0 {
		t.Fatalf("no writes expected when read fails")
	}

}

func TestRunOnce_RestartFailureReturnedAfterWrite(t *testing.T) {
	restartErr := errors.New("restart failure")
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true}
	store := &fakeStore{readData: []byte("existing"), recorder: &callRecorder{}}
	restarter := &fakeRestarter{err: restartErr}
	service := NewService(&fakeFetcher{}, rewriter, store, restarter, "pat", nil)

	result := service.RunOnce(context.Background())

	if !errors.Is(result.Err, restartErr) {
		t.Fatalf("expected restart error, got %v", result.Err)
	}
	if len(store.writes) != 1 {
		t.Fatalf("write should have been attempted before restart")
	}
	if !restarter.called {
		t.Fatalf("restart should have been called even if it failed")
	}
}

func TestRunOnce_PassesReadBytesToRewriter(t *testing.T) {
	input := []byte("current config")
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true}
	store := &fakeStore{readData: input}
	service := NewService(&fakeFetcher{}, rewriter, store, &fakeRestarter{restarted: []string{}}, "", nil)

	service.RunOnce(context.Background())

	if string(rewriter.lastCurrent) != string(input) {
		t.Fatalf("expected rewriter to receive read bytes, got %q", rewriter.lastCurrent)
	}
}

func TestRunOnce_ConvertsCloudflareServersForRewriter(t *testing.T) {
	servers := []cloudflare.ICEServer{{URLs: []string{"turn.example.com"}, Username: "u", Credential: "p"}}
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true}
	service := NewService(&fakeFetcher{servers: servers}, rewriter, &fakeStore{readData: []byte{}}, &fakeRestarter{restarted: []string{}}, "", nil)

	service.RunOnce(context.Background())

	if len(rewriter.lastServers) != len(servers) {
		t.Fatalf("expected %d servers, got %d", len(servers), len(rewriter.lastServers))
	}
	for i, server := range servers {
		rewritten := rewriter.lastServers[i]
		if len(rewritten.URLs) != len(server.URLs) || rewritten.URLs[0] != server.URLs[0] {
			t.Fatalf("urls mismatch at index %d", i)
		}
		if rewritten.Username != server.Username {
			t.Fatalf("username mismatch at index %d", i)
		}
		if rewritten.Credential != server.Credential {
			t.Fatalf("credential mismatch at index %d", i)
		}
	}
}

func TestRunOnce_ForwardsRestartArgs(t *testing.T) {
	pattern := "neko-*"
	timeout := 7 * time.Second
	rewriter := &fakeRewriter{result: []byte("updated"), changed: true}
	restarter := &fakeRestarter{restarted: []string{}}
	service := NewService(&fakeFetcher{}, rewriter, &fakeStore{readData: []byte{}}, restarter, pattern, &timeout)

	service.RunOnce(context.Background())

	if restarter.pattern != pattern {
		t.Fatalf("expected restart pattern %q, got %q", pattern, restarter.pattern)
	}
	if restarter.timeout == nil || *restarter.timeout != timeout {
		t.Fatalf("expected restart timeout %v, got %v", timeout, restarter.timeout)
	}
}

func TestRunOnce_SkipsWhenAlreadyRunning(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	service := NewService(&blockingFetcher{started: started, release: release}, &fakeRewriter{}, &fakeStore{}, &fakeRestarter{}, "", nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = service.RunOnce(context.Background())
	}()

	<-started
	result := service.RunOnce(context.Background())
	if !result.Busy || !result.Skipped {
		t.Fatalf("expected busy skipped result, got %+v", result)
	}
	if result.Err != nil {
		t.Fatalf("expected no error for busy skip, got %v", result.Err)
	}

	close(release)
	wg.Wait()
}

type callRecorder struct {
	ops []string
}

func (c *callRecorder) record(op string) {
	if c != nil {
		c.ops = append(c.ops, op)
	}
}

type fakeFetcher struct {
	servers  []cloudflare.ICEServer
	err      error
	recorder *callRecorder
}

type blockingFetcher struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (f *blockingFetcher) Fetch(ctx context.Context) ([]cloudflare.ICEServer, error) {
	select {
	case f.started <- struct{}{}:
	default:
	}
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return nil, nil
}

func (f *fakeFetcher) Fetch(ctx context.Context) ([]cloudflare.ICEServer, error) {
	if f.recorder != nil {
		f.recorder.record("fetch")
	}
	return f.servers, f.err
}

type fakeRewriter struct {
	result      []byte
	changed     bool
	err         error
	recorder    *callRecorder
	lastCurrent []byte
	lastServers []neko.ICEServer
	called      bool
}

func (f *fakeRewriter) Rewrite(ctx context.Context, current []byte, servers []neko.ICEServer) ([]byte, bool, error) {
	if f.recorder != nil {
		f.recorder.record("rewrite")
	}
	f.called = true
	f.lastCurrent = append([]byte(nil), current...)
	f.lastServers = make([]neko.ICEServer, len(servers))
	copy(f.lastServers, servers)
	return f.result, f.changed, f.err
}

type fakeStore struct {
	readData []byte
	readErr  error
	writeErr error
	writes   [][]byte
	recorder *callRecorder
}

func (f *fakeStore) Read(ctx context.Context) ([]byte, error) {
	if f.recorder != nil {
		f.recorder.record("read")
	}
	return f.readData, f.readErr
}

func (f *fakeStore) Write(ctx context.Context, data []byte) error {
	if f.recorder != nil {
		f.recorder.record("write")
	}
	f.writes = append(f.writes, append([]byte(nil), data...))
	return f.writeErr
}

type fakeRestarter struct {
	restarted []string
	err       error
	called    bool
	recorder  *callRecorder
	pattern   string
	timeout   *time.Duration
}

func (f *fakeRestarter) RestartMatching(ctx context.Context, pattern string, timeout *time.Duration) ([]string, error) {
	if f.recorder != nil {
		f.recorder.record("restart")
	}
	f.called = true
	f.pattern = pattern
	if timeout != nil {
		copyTimeout := *timeout
		f.timeout = &copyTimeout
	} else {
		f.timeout = nil
	}
	return append([]string(nil), f.restarted...), f.err
}
