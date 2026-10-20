package doctor

import (
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

func TestWorkContextDiagnostics(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // asserts the English text contract
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)
	cfg := config.Default()

	byName := func(checks []Check) map[string]Check {
		m := map[string]Check{}
		for _, c := range checks {
			m[c.Name] = c
		}
		return m
	}

	never := byName(workContextDiagnostics(cfg, workcontext.CalendarSnapshot{}, 0, now))
	if c := never["Calendar"]; c.Status != "info" || !strings.Contains(c.Detail, "menu bar app") {
		t.Fatalf("never synced = %+v", c)
	}
	if c := never["Meeting-aware breaks"]; c.Status != "info" || !strings.Contains(c.Detail, "off") {
		t.Fatalf("aware breaks = %+v", c)
	}

	fresh := workcontext.CalendarSnapshot{UpdatedAt: now.Add(-3 * time.Minute).Unix(), Events: make([]workcontext.Event, 4)}
	if c := byName(workContextDiagnostics(cfg, fresh, 7, now))["Calendar"]; c.Status != "ok" || !strings.Contains(c.Detail, "4 events") || !strings.Contains(c.Detail, "3 min ago") {
		t.Fatalf("fresh = %+v", c)
	}
	stale := workcontext.CalendarSnapshot{UpdatedAt: now.Add(-40 * time.Minute).Unix()}
	if c := byName(workContextDiagnostics(cfg, stale, 0, now))["Calendar"]; c.Status != "warn" {
		t.Fatalf("stale = %+v", c)
	}

	if c := byName(workContextDiagnostics(cfg, fresh, 7, now))["App usage"]; c.Status != "ok" || !strings.Contains(c.Detail, "7 apps") {
		t.Fatalf("usage = %+v", c)
	}
	cfg.AppUsageEnabled = false
	if c := byName(workContextDiagnostics(cfg, fresh, 0, now))["App usage"]; c.Status != "info" || !strings.Contains(c.Detail, "off") {
		t.Fatalf("usage off = %+v", c)
	}
}
