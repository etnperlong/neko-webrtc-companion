package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_LogsStructuredErrorOnLoggingConfigFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run(func(key string) string {
		if key == "LOG_LEVEL" {
			return "trace"
		}
		return ""
	}, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "failed to load logging config") {
		t.Fatalf("expected structured startup log, got %q", stderr.String())
	}
}
