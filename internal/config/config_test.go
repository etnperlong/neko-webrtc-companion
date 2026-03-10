package config

import (
	"testing"
	"time"
)

func TestLoadConfig_RequiresMandatoryFields(t *testing.T) {
	if _, err := LoadFromEnv(func(string) string { return "" }); err == nil {
		t.Fatalf("expected LoadFromEnv to return an error when mandatory env vars are missing")
	}
}

func TestLoadConfig_AppliesOptionalDefaults(t *testing.T) {
	env := map[string]string{
		envCron:                "0 0 * * *",
		envCloudflareTURNKeyID: "key",
		envCloudflareAPIToken:  "token",
		envNekoConfigPath:      "/tmp/config.yaml",
	}
	getter := func(key string) string {
		return env[key]
	}

	cfg, err := LoadFromEnv(getter)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.CloudflareTURNTTL != 86400 {
		t.Fatalf("expected TTL default 86400, got %d", cfg.CloudflareTURNTTL)
	}
	if !cfg.RunOnStart {
		t.Fatalf("expected RunOnStart default true, got false")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected HTTP addr default :8080, got %s", cfg.HTTPAddr)
	}
	if cfg.DockerContainerNameGlob != "neko-rooms-*" {
		t.Fatalf("expected default container glob neko-rooms-*, got %s", cfg.DockerContainerNameGlob)
	}
	if cfg.DockerImageGlob != "" {
		t.Fatalf("expected empty default image glob, got %s", cfg.DockerImageGlob)
	}
	if cfg.DockerLabelTrueKey != "" {
		t.Fatalf("expected empty default label key, got %s", cfg.DockerLabelTrueKey)
	}
}

func TestLoadConfig_OverridesOptionalValues(t *testing.T) {
	env := map[string]string{
		envCron:                    "0 0 * * *",
		envCloudflareTURNKeyID:     "key",
		envCloudflareAPIToken:      "token",
		envNekoConfigPath:          "/tmp/config.yaml",
		envHTTPAddr:                ":9090",
		envCloudflareTURNTTL:       "3600",
		envRunOnStart:              "false",
		envDockerContainerNameGlob: "custom-*",
		envDockerImageGlob:         "org/*",
		envDockerLabelTrueKey:      "true",
		envDockerRestartTimeout:    "5s",
	}
	getter := func(key string) string { return env[key] }

	cfg, err := LoadFromEnv(getter)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.CloudflareTURNTTL != 3600 {
		t.Fatalf("expected TTL override, got %d", cfg.CloudflareTURNTTL)
	}
	if cfg.RunOnStart {
		t.Fatalf("expected RunOnStart override to false, got true")
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected HTTP addr override :9090, got %s", cfg.HTTPAddr)
	}
	if cfg.DockerContainerNameGlob != "custom-*" {
		t.Fatalf("expected custom container glob, got %s", cfg.DockerContainerNameGlob)
	}
	if cfg.DockerImageGlob != "org/*" {
		t.Fatalf("expected custom image glob, got %s", cfg.DockerImageGlob)
	}
	if cfg.DockerLabelTrueKey != "true" {
		t.Fatalf("expected custom label key, got %s", cfg.DockerLabelTrueKey)
	}
	if cfg.DockerRestartTimeout != 5*time.Second {
		t.Fatalf("expected restart timeout 5s, got %v", cfg.DockerRestartTimeout)
	}
}

func TestLoadConfig_InvalidRunOnStart(t *testing.T) {
	env := map[string]string{
		envCron:                "0 0 * * *",
		envCloudflareTURNKeyID: "key",
		envCloudflareAPIToken:  "token",
		envNekoConfigPath:      "/tmp/config.yaml",
		envRunOnStart:          "not-bool",
	}
	getter := func(key string) string { return env[key] }
	if _, err := LoadFromEnv(getter); err == nil {
		t.Fatalf("expected error parsing %s", envRunOnStart)
	}
}

func TestLoadConfig_InvalidTTL(t *testing.T) {
	env := map[string]string{
		envCron:                "0 0 * * *",
		envCloudflareTURNKeyID: "key",
		envCloudflareAPIToken:  "token",
		envNekoConfigPath:      "/tmp/config.yaml",
		envCloudflareTURNTTL:   "not-int",
	}
	getter := func(key string) string { return env[key] }
	if _, err := LoadFromEnv(getter); err == nil {
		t.Fatalf("expected error parsing %s", envCloudflareTURNTTL)
	}
}

func TestLoadConfig_InvalidDockerRestartTimeout(t *testing.T) {
	env := map[string]string{
		envCron:                 "0 0 * * *",
		envCloudflareTURNKeyID:  "key",
		envCloudflareAPIToken:   "token",
		envNekoConfigPath:       "/tmp/config.yaml",
		envDockerRestartTimeout: "not-duration",
	}
	getter := func(key string) string { return env[key] }
	if _, err := LoadFromEnv(getter); err == nil {
		t.Fatalf("expected error parsing %s", envDockerRestartTimeout)
	}
}
