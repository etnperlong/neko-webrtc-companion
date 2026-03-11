package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	kenv "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	envConfigFile              = "CONFIG_FILE"
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

var envKeyMap = map[string]string{
	envCron:                    "cron",
	envCloudflareTURNKeyID:     "cloudflare_turn_key_id",
	envCloudflareAPIToken:      "cloudflare_api_token",
	envNekoConfigPath:          "neko_config_path",
	envHTTPAddr:                "http_addr",
	envCloudflareTURNTTL:       "cloudflare_turn_ttl",
	envRunOnStart:              "run_on_start",
	envDockerContainerNameGlob: "docker_container_name_glob",
	envDockerImageGlob:         "docker_image_glob",
	envDockerLabelTrueKey:      "docker_label_true_key",
	envDockerRestartTimeout:    "docker_restart_timeout",
}

// Config holds runtime configuration values for the turn refresh service.
type Config struct {
	Cron                    string        `koanf:"cron"`
	CloudflareTURNKeyID     string        `koanf:"cloudflare_turn_key_id"`
	CloudflareAPIToken      string        `koanf:"cloudflare_api_token"`
	NekoConfigPath          string        `koanf:"neko_config_path"`
	HTTPAddr                string        `koanf:"http_addr"`
	CloudflareTURNTTL       int           `koanf:"cloudflare_turn_ttl"`
	RunOnStart              bool          `koanf:"run_on_start"`
	DockerContainerNameGlob string        `koanf:"docker_container_name_glob"`
	DockerImageGlob         string        `koanf:"docker_image_glob"`
	DockerLabelTrueKey      string        `koanf:"docker_label_true_key"`
	DockerRestartTimeout    time.Duration `koanf:"docker_restart_timeout"`
}

// Load loads configuration from defaults, an optional YAML config file, and environment variables.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	k := koanf.New(".")
	if err := k.Load(confmap.Provider(defaultConfig(), "."), nil); err != nil {
		return Config{}, fmt.Errorf("load default config: %w", err)
	}

	configFile := strings.TrimSpace(getenv(envConfigFile))
	if configFile != "" {
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return Config{}, fmt.Errorf("load %s %q: %w", envConfigFile, configFile, err)
		}
	}

	if err := k.Load(kenv.Provider(".", kenv.Opt{
		TransformFunc: transformEnv,
		EnvironFunc: func() []string {
			return environFromGetter(getenv)
		},
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load env config: %w", err)
	}

	cfg, err := decodeConfig(k)
	if err != nil {
		return Config{}, err
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadFromEnv loads configuration from environment variables using the provided getenv function.
func LoadFromEnv(getenv func(string) string) (Config, error) {
	return Load(getenv)
}

func defaultConfig() map[string]any {
	return map[string]any{
		"http_addr":                  ":8080",
		"cloudflare_turn_ttl":        defaultCloudflareTURNTTL,
		"run_on_start":               true,
		"docker_container_name_glob": defaultDockerContainerNameGlob,
	}
}

func environFromGetter(getenv func(string) string) []string {
	keys := []string{
		envConfigFile,
		envCron,
		envCloudflareTURNKeyID,
		envCloudflareAPIToken,
		envNekoConfigPath,
		envHTTPAddr,
		envCloudflareTURNTTL,
		envRunOnStart,
		envDockerContainerNameGlob,
		envDockerImageGlob,
		envDockerLabelTrueKey,
		envDockerRestartTimeout,
	}

	environ := make([]string, 0, len(keys))
	for _, key := range keys {
		environ = append(environ, key+"="+getenv(key))
	}

	return environ
}

func transformEnv(key, value string) (string, any) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	configKey, ok := envKeyMap[key]
	if !ok {
		return "", nil
	}

	return configKey, trimmed
}

func decodeConfig(k *koanf.Koanf) (Config, error) {
	var cfg Config

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "koanf",
		Result:           &cfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToBasicTypeHookFunc(),
		),
	})
	if err != nil {
		return Config{}, fmt.Errorf("build config decoder: %w", err)
	}

	if err := decoder.Decode(k.All()); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	var missing []string

	if strings.TrimSpace(cfg.Cron) == "" {
		missing = append(missing, envCron)
	}
	if strings.TrimSpace(cfg.CloudflareTURNKeyID) == "" {
		missing = append(missing, envCloudflareTURNKeyID)
	}
	if strings.TrimSpace(cfg.CloudflareAPIToken) == "" {
		missing = append(missing, envCloudflareAPIToken)
	}
	if strings.TrimSpace(cfg.NekoConfigPath) == "" {
		missing = append(missing, envNekoConfigPath)
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return nil
}
