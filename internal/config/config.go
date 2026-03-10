package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	envCron                = "NEKO_TURN_CRON"
	envCloudflareTURNKeyID = "CLOUDFLARE_TURN_KEY_ID"
	envCloudflareAPIToken  = "CLOUDFLARE_API_TOKEN"
	envNekoConfigPath      = "NEKO_CONFIG_PATH"
	envHTTPAddr            = "HTTP_ADDR"
)

// Config holds runtime configuration values for the turn refresh service.
type Config struct {
	Cron                string
	CloudflareTURNKeyID string
	CloudflareAPIToken  string
	NekoConfigPath      string
	HTTPAddr            string
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

	return Config{
		Cron:                cron,
		CloudflareTURNKeyID: cfKeyID,
		CloudflareAPIToken:  cfToken,
		NekoConfigPath:      nekoConfig,
		HTTPAddr:            httpAddr,
	}, nil
}
