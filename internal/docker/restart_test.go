package docker

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
)

func TestMatchContainerNames_ReturnsGlobMatches(t *testing.T) {
	matches, err := MatchContainerNames([]string{"neko-rooms-a", "db", "neko-rooms-b"}, "neko-rooms-*")
	if err != nil {
		t.Fatalf("unexpected error matching containers: %v", err)
	}

	expected := []string{"neko-rooms-a", "neko-rooms-b"}
	if !reflect.DeepEqual(matches, expected) {
		t.Fatalf("expected matches %v, got %v", expected, matches)
	}
}

func TestMatchContainerNames_TrimsLeadingSlash(t *testing.T) {
	matches, err := MatchContainerNames([]string{"/neko-rooms-a", "db", "/neko-rooms-b"}, "neko-rooms-*")
	if err != nil {
		t.Fatalf("unexpected error matching containers: %v", err)
	}

	expected := []string{"neko-rooms-a", "neko-rooms-b"}
	if !reflect.DeepEqual(matches, expected) {
		t.Fatalf("expected matches %v, got %v", expected, matches)
	}
}

func TestMatchContainerNames_InvalidPattern(t *testing.T) {
	if _, err := MatchContainerNames([]string{"foo"}, "["); err == nil {
		t.Fatalf("expected error for invalid pattern, got nil")
	}
}

func TestRestarter_RestartsMatchingContainers(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/neko-rooms-a"}, Image: "neko:latest", Labels: map[string]string{"managed": "true"}},
			{ID: "c2", Names: []string{"/db"}, Image: "postgres:latest", Labels: map[string]string{"managed": "true"}},
			{ID: "c3", Names: []string{"/neko-rooms-b"}, Image: "neko:latest", Labels: map[string]string{"managed": "true"}},
		},
	}

	restarter := NewRestarter(fake, ContainerFilters{})
	timeout := 5 * time.Second

	restarted, err := restarter.RestartMatching(context.Background(), "neko-rooms-*", &timeout)
	if err != nil {
		t.Fatalf("unexpected error restarting containers: %v", err)
	}

	expectedNames := []string{"neko-rooms-a", "neko-rooms-b"}
	if !reflect.DeepEqual(restarted, expectedNames) {
		t.Fatalf("expected restarted names %v, got %v", expectedNames, restarted)
	}

	if !reflect.DeepEqual(fake.restartedIDs, []string{"c1", "c3"}) {
		t.Fatalf("expected restarted IDs [c1 c3], got %v", fake.restartedIDs)
	}

	expectedTimeout := stopTimeout(&timeout)
	if expectedTimeout == nil {
		t.Fatalf("expected helper to produce timeout for %v", timeout)
	}

	for _, observed := range fake.observedOpts {
		if observed.Timeout == nil {
			t.Fatalf("expected timeout to be forwarded, got nil")
		}
		if *observed.Timeout != *expectedTimeout {
			t.Fatalf("expected timeout %v, got %v", *expectedTimeout, *observed.Timeout)
		}
	}
}

func TestRestarter_RestartMatching_NoMatches(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{{ID: "c1", Names: []string{"/db"}, Image: "postgres:latest", Labels: map[string]string{"managed": "true"}}},
	}

	restarter := NewRestarter(fake, ContainerFilters{})

	restarted, err := restarter.RestartMatching(context.Background(), "neko-rooms-*", nil)
	if err != nil {
		t.Fatalf("unexpected error restarting containers: %v", err)
	}

	if len(restarted) != 0 {
		t.Fatalf("expected no restarted containers, got %v", restarted)
	}

	if len(fake.restartedIDs) != 0 {
		t.Fatalf("expected no restart calls, got %v", fake.restartedIDs)
	}
}

func TestRestarter_ContainerListError(t *testing.T) {
	listErr := errors.New("boom")
	fake := &fakeClient{listErr: listErr}

	restarter := NewRestarter(fake, ContainerFilters{})
	pattern := "neko-rooms-*"

	_, err := restarter.RestartMatching(context.Background(), pattern, nil)
	if err == nil {
		t.Fatalf("expected error from list failure")
	}

	if !strings.Contains(err.Error(), "list containers") || !strings.Contains(err.Error(), pattern) {
		t.Fatalf("unexpected error context: %v", err)
	}
}

func TestRestarter_ContainerRestartError(t *testing.T) {
	restartErr := errors.New("boom")
	fake := &fakeClient{
		containers: []types.Container{{ID: "c1", Names: []string{"/neko-rooms-a"}, Image: "neko:latest", Labels: map[string]string{"managed": "true"}}},
		restartErr: restartErr,
	}

	restarter := NewRestarter(fake, ContainerFilters{})
	pattern := "neko-rooms-*"

	_, err := restarter.RestartMatching(context.Background(), pattern, nil)
	if err == nil {
		t.Fatalf("expected error from restart failure")
	}

	if !strings.Contains(err.Error(), "restart container c1") || !strings.Contains(err.Error(), pattern) {
		t.Fatalf("unexpected error context: %v", err)
	}
}

func TestRestarter_MatchContainerNamesError(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{{ID: "c1", Names: []string{"/neko-rooms-a"}, Image: "neko:latest", Labels: map[string]string{"managed": "true"}}},
	}

	restarter := NewRestarter(fake, ContainerFilters{})
	pattern := "["

	_, err := restarter.RestartMatching(context.Background(), pattern, nil)
	if err == nil {
		t.Fatalf("expected error from invalid glob pattern")
	}

	if !strings.Contains(err.Error(), "match runtime pattern") || !strings.Contains(err.Error(), pattern) {
		t.Fatalf("unexpected error context: %v", err)
	}
}

func TestStopTimeout_RoundsUp(t *testing.T) {
	duration := 1500 * time.Millisecond
	result := stopTimeout(&duration)
	if result == nil {
		t.Fatalf("expected timeout conversion to return a value for %v", duration)
	}
	if *result != 2 {
		t.Fatalf("expected rounded timeout 2 seconds, got %v", *result)
	}
}

func TestStopTimeout_NonPositiveBecomesNil(t *testing.T) {
	zero := time.Duration(0)
	if got := stopTimeout(&zero); got != nil {
		t.Fatalf("expected nil timeout for zero duration, got %v", got)
	}

	negative := -1 * time.Second
	if got := stopTimeout(&negative); got != nil {
		t.Fatalf("expected nil timeout for negative duration, got %v", got)
	}
}

func TestRestarter_MatchesNameImageAndLabelWithANDSemantics(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/neko-rooms-a"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "true"}},
			{ID: "c2", Names: []string{"/neko-rooms-b"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "false"}},
			{ID: "c3", Names: []string{"/neko-rooms-c"}, Image: "busybox:latest", Labels: map[string]string{"turn.managed": "true"}},
			{ID: "c4", Names: []string{"/db"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "true"}},
		},
	}

	restarter := NewRestarter(fake, ContainerFilters{
		NamePattern:  "neko-rooms-*",
		ImagePattern: "ghcr.io/m1k1o/neko:*",
		LabelTrueKey: "turn.managed",
	})

	restarted, err := restarter.RestartMatching(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error restarting containers: %v", err)
	}

	expectedNames := []string{"neko-rooms-a"}
	if !reflect.DeepEqual(restarted, expectedNames) {
		t.Fatalf("expected restarted names %v, got %v", expectedNames, restarted)
	}

	if !reflect.DeepEqual(fake.restartedIDs, []string{"c1"}) {
		t.Fatalf("expected restarted IDs [c1], got %v", fake.restartedIDs)
	}
}

func TestRestarter_RuntimePatternAndConfiguredNamePatternBothApply(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/neko-prod-a"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "true"}},
			{ID: "c2", Names: []string{"/neko-dev-a"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "true"}},
		},
	}

	restarter := NewRestarter(fake, ContainerFilters{NamePattern: "neko-prod-*"})

	restarted, err := restarter.RestartMatching(context.Background(), "neko-*", nil)
	if err != nil {
		t.Fatalf("unexpected error restarting containers: %v", err)
	}

	expected := []string{"neko-prod-a"}
	if !reflect.DeepEqual(restarted, expected) {
		t.Fatalf("expected restarted names %v, got %v", expected, restarted)
	}
	if !reflect.DeepEqual(fake.restartedIDs, []string{"c1"}) {
		t.Fatalf("expected restarted IDs [c1], got %v", fake.restartedIDs)
	}
}

func TestRestarter_MultiNameContainerReportedOnce(t *testing.T) {
	fake := &fakeClient{
		containers: []types.Container{
			{ID: "c1", Names: []string{"/neko-rooms-a", "/alias-a"}, Image: "ghcr.io/m1k1o/neko:latest", Labels: map[string]string{"turn.managed": "true"}},
		},
	}

	restarter := NewRestarter(fake, ContainerFilters{NamePattern: "*a"})

	restarted, err := restarter.RestartMatching(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error restarting containers: %v", err)
	}

	if len(restarted) != 1 {
		t.Fatalf("expected one reported restart, got %v", restarted)
	}
	if !reflect.DeepEqual(fake.restartedIDs, []string{"c1"}) {
		t.Fatalf("expected restarted IDs [c1], got %v", fake.restartedIDs)
	}
}

type fakeClient struct {
	containers []types.Container

	restartErr error
	listErr    error

	restartedIDs []string
	observedOpts []containertypes.StopOptions
}

func (f *fakeClient) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	return f.containers, f.listErr
}

func (f *fakeClient) ContainerRestart(_ context.Context, containerID string, options containertypes.StopOptions) error {
	f.restartedIDs = append(f.restartedIDs, containerID)
	f.observedOpts = append(f.observedOpts, options)
	return f.restartErr
}
