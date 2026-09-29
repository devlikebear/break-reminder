package doctor

import "github.com/devlikebear/break-reminder/internal/i18n"

import "github.com/devlikebear/break-reminder/internal/timetools"

func timeToolsDiagnostic(status string, runtime timetools.Runtime, now int64) Check {
	check := Check{Name: i18n.Text("Time tools worker"), Status: "warn"}
	if runtime.Error != "" {
		check.Status = "fail"
		check.Detail = runtime.Error + i18n.Text("; preserve time-tools.json and repair, then run 'service start'")
		return check
	}
	if status == "Not Installed" {
		check.Detail = i18n.Text("not installed; after upgrading run 'break-reminder service install' once")
		return check
	}
	if !runtime.Running(now) {
		check.Detail = i18n.Text(status) + i18n.Text("; heartbeat unavailable/stale; run 'break-reminder service start'")
		return check
	}
	check.Status = "ok"
	check.Detail = i18n.Text("running; independent countdown worker")
	if !runtime.NotificationAvailable {
		check.Status = "warn"
		check.Detail += i18n.Text("; terminal-notifier unavailable: completion remains in menu/dashboard, sound is not guaranteed; install terminal-notifier and restart service")
	}
	return check
}
