package docker

import (
	"context"
	"fmt"
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
	client Client
}

// NewRestarter creates a Restarter that uses the provided Docker client.
func NewRestarter(client Client) *Restarter {
	return &Restarter{client: client}
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

// RestartMatching restarts every container whose name matches the given pattern.
// It returns the normalized container names that were restarted.
func (r *Restarter) RestartMatching(ctx context.Context, pattern string, timeout *time.Duration) ([]string, error) {
	containers, err := r.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list containers for pattern %q: %w", pattern, err)
	}

	var restarted []string
	for _, container := range containers {
		matches, err := MatchContainerNames(container.Names, pattern)
		if err != nil {
			return nil, fmt.Errorf("match containers %s for pattern %q: %w", container.ID, pattern, err)
		}
		if len(matches) == 0 {
			continue
		}

		stopOptions := containertypes.StopOptions{Timeout: stopTimeout(timeout)}
		if err := r.client.ContainerRestart(ctx, container.ID, stopOptions); err != nil {
			return nil, fmt.Errorf("restart container %s for pattern %q: %w", container.ID, pattern, err)
		}

		restarted = append(restarted, matches...)
	}

	return restarted, nil
}

func stopTimeout(timeout *time.Duration) *int {
	if timeout == nil {
		return nil
	}

	seconds := int(math.Ceil(timeout.Seconds()))
	if seconds == 0 && *timeout > 0 {
		seconds = 1
	}
	return &seconds
}
