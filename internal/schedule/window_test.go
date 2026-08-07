package schedule

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func TestEvaluateNoWindowsAllows(t *testing.T) {
	decision, err := Evaluate(time.Now(), nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected allowed with no windows")
	}
}

func TestEvaluateAllowWindowActive(t *testing.T) {
	now := mustTime(t, "2026-08-06T22:55:30+02:00")
	windows := []Window{{Allow: true, Schedule: "55 22 * * *", Duration: "1h", TimeZone: "Europe/Berlin"}}
	decision, err := Evaluate(now, windows)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected allowed inside active allow window")
	}
}

func TestEvaluateAllowWindowClosedBlocksWithRequeue(t *testing.T) {
	now := mustTime(t, "2026-08-06T22:40:00+02:00")
	windows := []Window{{Allow: true, Schedule: "55 22 * * *", Duration: "1h", TimeZone: "Europe/Berlin"}}
	decision, err := Evaluate(now, windows)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected blocked before allow window opens")
	}
	if decision.RequeueAt == nil {
		t.Fatalf("expected requeue time toward window open")
	}
}

func TestEvaluateDenyWindowActiveBlocks(t *testing.T) {
	now := mustTime(t, "2026-08-06T22:55:30+02:00")
	windows := []Window{
		{Allow: true, Schedule: "* * * * *", Duration: "1h", TimeZone: "Europe/Berlin"},
		{Allow: false, Schedule: "55 22 * * *", Duration: "1h", TimeZone: "Europe/Berlin"},
	}
	decision, err := Evaluate(now, windows)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected active deny window to block even with an active allow window")
	}
}

func TestEvaluateInvalidScheduleErrors(t *testing.T) {
	_, err := Evaluate(time.Now(), []Window{{Allow: true, Schedule: "not-a-cron", Duration: "1h"}})
	if err == nil {
		t.Fatalf("expected error for invalid cron schedule")
	}
}
