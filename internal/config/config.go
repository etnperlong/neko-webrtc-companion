package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envCron                    = "NEKO_TURN_CRON"
	envCloudflareTURNKeyID     = "CLOUDFLARE_TURN_KEY_ID"
	envCloudflareAPIToken      = "CLOUDFLARE_API_TOKEN"
	envNekoConfigPath          = "NEKO_CONFIG_PATH"
	envHTTPAddr                = "HTTP_ADDR"
	envCloudflareTURNTTL       = "CLOUDFLARE_TURN_TTL"
	envRunOnStart              = "RUN_ON_START"
	envDockerContainerNameGlob = "DOCKER_CONTAINER_NAME_GLOB"
	envDockerImageGlob         = "DOCKER_IMAGE_GLOB"
	envDockerLabelTrueKey      = "DOCKER_LABEL_TRUE_KEY"
	envDockerRestartTimeout    = "DOCKER_RESTART_TIMEOUT"
)

const (
	defaultCloudflareTURNTTL       = 86400
	defaultDockerContainerNameGlob = "neko-rooms-*"
)

// Config holds runtime configuration values for the turn refresh service.
type Config struct {
	Cron                    string
	CloudflareTURNKeyID     string
	CloudflareAPIToken      string
	NekoConfigPath          string
	HTTPAddr                string
	CloudflareTURNTTL       int
	RunOnStart              bool
	DockerContainerNameGlob string
	DockerImageGlob         string
	DockerLabelTrueKey      string
	DockerRestartTimeout    time.Duration
}

// LoadFromEnv loads configuration from environment variables using the provided getenv function.
func LoadFromEnv(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	var missing []string

	cron := strings.TrimSpace(getenv(envCron))
	if cron == "" {
		missing = append(missing, envCron)
	}

	cfKeyID := strings.TrimSpace(getenv(envCloudflareTURNKeyID))
	if cfKeyID == "" {
		missing = append(missing, envCloudflareTURNKeyID)
	}

	cfToken := strings.TrimSpace(getenv(envCloudflareAPIToken))
	if cfToken == "" {
		missing = append(missing, envCloudflareAPIToken)
	}

	nekoConfig := strings.TrimSpace(getenv(envNekoConfigPath))
	if nekoConfig == "" {
		missing = append(missing, envNekoConfigPath)
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	httpAddr := strings.TrimSpace(getenv(envHTTPAddr))
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	ttl := defaultCloudflareTURNTTL
	if raw := strings.TrimSpace(getenv(envCloudflareTURNTTL)); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envCloudflareTURNTTL, err)
		}
		ttl = parsed
	}

	runOnStart := true
	if raw := strings.TrimSpace(getenv(envRunOnStart)); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envRunOnStart, err)
		}
		runOnStart = parsed
	}

	dockerContainerNameGlob := strings.TrimSpace(getenv(envDockerContainerNameGlob))
	if dockerContainerNameGlob == "" {
		dockerContainerNameGlob = defaultDockerContainerNameGlob
	}

	dockerImageGlob := strings.TrimSpace(getenv(envDockerImageGlob))
	dockerLabelTrueKey := strings.TrimSpace(getenv(envDockerLabelTrueKey))

	var dockerRestartTimeout time.Duration
	if raw := strings.TrimSpace(getenv(envDockerRestartTimeout)); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envDockerRestartTimeout, err)
		}
		dockerRestartTimeout = parsed
	}

	return Config{
		Cron:                    cron,
		CloudflareTURNKeyID:     cfKeyID,
		CloudflareAPIToken:      cfToken,
		NekoConfigPath:          nekoConfig,
		HTTPAddr:                httpAddr,
		CloudflareTURNTTL:       ttl,
		RunOnStart:              runOnStart,
		DockerContainerNameGlob: dockerContainerNameGlob,
		DockerImageGlob:         dockerImageGlob,
		DockerLabelTrueKey:      dockerLabelTrueKey,
		DockerRestartTimeout:    dockerRestartTimeout,
	}, nil
}
