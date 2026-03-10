package scheduler

import (
	"context"
	"testing"
)

func TestNewScheduler_RejectsInvalidCron(t *testing.T) {
	if _, err := New("not-a-cron", func(context.Context) {}); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}
