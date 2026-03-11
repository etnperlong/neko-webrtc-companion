package docker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

// Client defines the operations from the Docker SDK that the restarter needs.
type Client interface {
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerRestart(ctx context.Context, containerID string, options containertypes.StopOptions) error
}

// Restarter handles restarting containers whose names match a glob.
type Restarter struct {
	client  Client
	filters ContainerFilters
}

// NewRestarter creates a Restarter that uses the provided Docker client.
func NewRestarter(client Client, filters ContainerFilters) *Restarter {
	return &Restarter{client: client, filters: filters}
}

// NewDockerClient creates a Docker client using environmental configuration.
func NewDockerClient() (Client, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &dockerclientWrapper{inner: cli}, nil
}

type dockerclientWrapper struct {
	inner *dockerclient.Client
}

func (d *dockerclientWrapper) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return d.inner.ContainerList(ctx, options)
}

func (d *dockerclientWrapper) ContainerRestart(ctx context.Context, containerID string, options containertypes.StopOptions) error {
	return d.inner.ContainerRestart(ctx, containerID, options)
}

// RestartMatching restarts every container whose configured filters match.
// It returns the normalized container names that were restarted.
func (r *Restarter) RestartMatching(ctx context.Context, pattern string, timeout *time.Duration) ([]string, error) {
	slog.Debug("docker restart scan starting", "component", "docker", "pattern", pattern, "name_pattern", r.filters.NamePattern, "image_pattern", r.filters.ImagePattern, "label_key", r.filters.LabelTrueKey)
	containers, err := r.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		slog.Error("docker container list failed", "component", "docker", "pattern", pattern, "err", err)
		return nil, fmt.Errorf("list containers for pattern %q: %w", pattern, err)
	}

	var restarted []string
	for _, container := range containers {
		filters := r.filters
		matches, err := MatchContainerNames(container.Names, filters.NamePattern)
		if err != nil {
			return nil, fmt.Errorf("match containers %s for pattern %q: %w", container.ID, filters.NamePattern, err)
		}
		if len(matches) == 0 {
			continue
		}

		if pattern != "" {
			matches, err = MatchContainerNames(matches, pattern)
			if err != nil {
				return nil, fmt.Errorf("match runtime pattern %q for container %s: %w", pattern, container.ID, err)
			}
		}

		if len(matches) == 0 || !hasLabelTrueValue(container.Labels, filters.LabelTrueKey) {
			continue
		}

		imageMatched, err := matchesImage(container.Image, filters.ImagePattern)
		if err != nil {
			return nil, fmt.Errorf("match image %q for container %s: %w", container.Image, container.ID, err)
		}
		if !imageMatched {
			continue
		}

		stopOptions := containertypes.StopOptions{Timeout: stopTimeout(timeout)}
		effectivePattern := filters.NamePattern
		if pattern != "" {
			effectivePattern = pattern
		}
		if err := r.client.ContainerRestart(ctx, container.ID, stopOptions); err != nil {
			slog.Error("docker container restart failed", "component", "docker", "container_id", container.ID, "pattern", effectivePattern, "err", err)
			return nil, fmt.Errorf("restart container %s for pattern %q: %w", container.ID, effectivePattern, err)
		}

		restarted = append(restarted, matches[0])
	}

	slog.Info("docker restart summary", "component", "docker", "restart_count", len(restarted), "containers", restarted)

	return restarted, nil
}

func stopTimeout(timeout *time.Duration) *int {
	if timeout == nil {
		return nil
	}
	if *timeout <= 0 {
		return nil
	}

	seconds := int(math.Ceil(timeout.Seconds()))
	if seconds == 0 {
		seconds = 1
	}
	return &seconds
}
