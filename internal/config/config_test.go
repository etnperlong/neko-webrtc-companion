package config

import "testing"

func TestLoadConfig_RequiresMandatoryFields(t *testing.T) {
	if _, err := LoadFromEnv(func(string) string { return "" }); err == nil {
		t.Fatalf("expected LoadFromEnv to return an error when mandatory env vars are missing")
	}
}
