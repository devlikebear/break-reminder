package doctor

import (
	"fmt"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/paths"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// workContextDiagnostics reports what the menu bar app last sent. Calendar
// permission itself lives in the app (EventKit), so the CLI judges by the
// last update it can see.
func workContextDiagnostics(cfg config.Config, snap workcontext.CalendarSnapshot, appsToday int, now time.Time) []Check {
	var checks []Check
	calendar := i18n.Text("Calendar")
	switch {
	case snap.UpdatedAt == 0:
		checks = append(checks, Check{calendar, "info", i18n.Text("no data yet (connect the calendar in the menu bar app)")})
	case snap.Stale(now):
		checks = append(checks, Check{calendar, "warn", fmt.Sprintf(i18n.Text("last update %d min ago; is the menu bar app running with calendar access?"), int(now.Sub(time.Unix(snap.UpdatedAt, 0)).Minutes()))})
	default:
		checks = append(checks, Check{calendar, "ok", fmt.Sprintf(i18n.Text("%d events, updated %d min ago"), len(snap.Events), int(now.Sub(time.Unix(snap.UpdatedAt, 0)).Minutes()))})
	}

	aware := i18n.Text("off")
	if cfg.CalendarAwareBreaks {
		aware = fmt.Sprintf(i18n.Text("on (meeting rule: %s)"), cfg.MeetingRule)
	}
	checks = append(checks, Check{i18n.Text("Meeting-aware breaks"), "info", aware})

	usage := i18n.Text("App usage")
	if !cfg.AppUsageEnabled {
		checks = append(checks, Check{usage, "info", i18n.Text("off")})
	} else {
		checks = append(checks, Check{usage, "ok", fmt.Sprintf(i18n.Text("%d apps recorded today"), appsToday)})
	}
	return checks
}

func workContextChecks(cfg config.Config, now time.Time) []Check {
	snap, _ := workcontext.LoadCalendar(paths.CalendarCacheFile())
	usage, _ := workcontext.UsageStore{Dir: paths.AppUsageDir()}.Day(now)
	return workContextDiagnostics(cfg, snap, len(usage), now)
}
