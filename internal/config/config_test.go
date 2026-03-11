package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testEnvConfigFile = "CONFIG_FILE"

func testEnvGetter(env map[string]string) func(string) string {
	return func(key string) string {
		return env[key]
	}
}

func writeTestConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

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

func TestLoadConfig_LoadsValuesFromYAMLFile(t *testing.T) {
	path := writeTestConfigFile(t, strings.TrimSpace(`
cron: "0 */6 * * *"
cloudflare_turn_key_id: "yaml-key"
cloudflare_api_token: "yaml-token"
neko_config_path: "/data/neko.yaml"
http_addr: ":9090"
cloudflare_turn_ttl: 3600
run_on_start: false
docker_container_name_glob: "yaml-*"
docker_image_glob: "ghcr.io/example/*"
docker_label_true_key: "managed"
docker_restart_timeout: "5s"
`))

	cfg, err := LoadFromEnv(testEnvGetter(map[string]string{
		testEnvConfigFile: path,
	}))
	if err != nil {
		t.Fatalf("unexpected error loading config from file: %v", err)
	}

	if cfg.Cron != "0 */6 * * *" {
		t.Fatalf("expected cron from file, got %q", cfg.Cron)
	}
	if cfg.CloudflareTURNKeyID != "yaml-key" {
		t.Fatalf("expected key id from file, got %q", cfg.CloudflareTURNKeyID)
	}
	if cfg.CloudflareAPIToken != "yaml-token" {
		t.Fatalf("expected token from file, got %q", cfg.CloudflareAPIToken)
	}
	if cfg.NekoConfigPath != "/data/neko.yaml" {
		t.Fatalf("expected neko config path from file, got %q", cfg.NekoConfigPath)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected http addr from file, got %q", cfg.HTTPAddr)
	}
	if cfg.CloudflareTURNTTL != 3600 {
		t.Fatalf("expected ttl from file, got %d", cfg.CloudflareTURNTTL)
	}
	if cfg.RunOnStart {
		t.Fatalf("expected run_on_start false from file")
	}
	if cfg.DockerContainerNameGlob != "yaml-*" {
		t.Fatalf("expected container glob from file, got %q", cfg.DockerContainerNameGlob)
	}
	if cfg.DockerImageGlob != "ghcr.io/example/*" {
		t.Fatalf("expected image glob from file, got %q", cfg.DockerImageGlob)
	}
	if cfg.DockerLabelTrueKey != "managed" {
		t.Fatalf("expected label key from file, got %q", cfg.DockerLabelTrueKey)
	}
	if cfg.DockerRestartTimeout != 5*time.Second {
		t.Fatalf("expected restart timeout 5s from file, got %v", cfg.DockerRestartTimeout)
	}
}

func TestLoadConfig_EnvironmentOverridesYAMLFile(t *testing.T) {
	path := writeTestConfigFile(t, strings.TrimSpace(`
cron: "0 */6 * * *"
cloudflare_turn_key_id: "yaml-key"
cloudflare_api_token: "yaml-token"
neko_config_path: "/data/neko.yaml"
http_addr: ":9090"
run_on_start: true
cloudflare_turn_ttl: 3600
`))

	cfg, err := LoadFromEnv(testEnvGetter(map[string]string{
		testEnvConfigFile:     path,
		envHTTPAddr:           " :8081 ",
		envRunOnStart:         "false",
		envCloudflareTURNTTL:  "7200",
		envCloudflareAPIToken: "override-token",
	}))
	if err != nil {
		t.Fatalf("unexpected error loading merged config: %v", err)
	}

	if cfg.HTTPAddr != ":8081" {
		t.Fatalf("expected env http addr override, got %q", cfg.HTTPAddr)
	}
	if cfg.RunOnStart {
		t.Fatalf("expected env run_on_start override to false")
	}
	if cfg.CloudflareTURNTTL != 7200 {
		t.Fatalf("expected env ttl override, got %d", cfg.CloudflareTURNTTL)
	}
	if cfg.CloudflareAPIToken != "override-token" {
		t.Fatalf("expected env token override, got %q", cfg.CloudflareAPIToken)
	}
}

func TestLoadConfig_ExplicitMissingConfigFileFails(t *testing.T) {
	_, err := LoadFromEnv(testEnvGetter(map[string]string{
		testEnvConfigFile: "/tmp/does-not-exist-config.yaml",
	}))
	if err == nil {
		t.Fatalf("expected missing config file to fail")
	}
}
