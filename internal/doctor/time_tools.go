package doctor

import "github.com/devlikebear/break-reminder/internal/timetools"

func timeToolsDiagnostic(status string, runtime timetools.Runtime, now int64) Check {
	check := Check{Name: "Time tools worker", Status: "warn"}
	if runtime.Error != "" {
		check.Status = "fail"
		check.Detail = runtime.Error + "; preserve time-tools.json and repair, then run 'service start'"
		return check
	}
	if status == "Not Installed" {
		check.Detail = "not installed; after upgrading run 'break-reminder service install' once"
		return check
	}
	if !runtime.Running(now) {
		check.Detail = status + "; heartbeat unavailable/stale; run 'break-reminder service start'"
		return check
	}
	check.Status = "ok"
	check.Detail = "running; independent countdown worker"
	if !runtime.NotificationAvailable {
		check.Status = "warn"
		check.Detail += "; terminal-notifier unavailable: completion remains in menu/dashboard, sound is not guaranteed; install terminal-notifier and restart service"
	}
	return check
}
