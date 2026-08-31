package schedule

import (
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

// IsWorkDay checks if the given time falls on a configured working day.
func IsWorkDay(cfg config.Config, t time.Time) bool {
	// Check day: time.Weekday() returns 0=Sun..6=Sat
	// Config uses ISO: 1=Mon..7=Sun
	isoDay := int(t.Weekday())
	if isoDay == 0 {
		isoDay = 7
	}

	for _, d := range cfg.WorkDays {
		if d == isoDay {
			return true
		}
	}
	return false
}

// IsWorkingTime checks if the given time falls within configured working hours and days.
func IsWorkingTime(cfg config.Config, t time.Time) bool {
	if !IsWorkDay(cfg, t) {
		return false
	}

	currentMinute := t.Hour()*60 + t.Minute()
	workStartMinute := cfg.WorkStartHour*60 + cfg.WorkStartMinute
	workEndMinute := cfg.WorkEndHour*60 + cfg.WorkEndMinute

	return currentMinute >= workStartMinute && currentMinute < workEndMinute
}

// InDetectWindow reports whether automatic work-session detection may start a
// session at the given time. The window is wider than the configured working
// hours so an early start or a late finish is still picked up.
func InDetectWindow(cfg config.Config, t time.Time) bool {
	if !IsWorkDay(cfg, t) {
		return false
	}
	return t.Hour() >= cfg.SessionDetectStartHour && t.Hour() < cfg.SessionDetectEndHour
}

// IsActive reports whether the timer is currently supposed to run. With
// automatic session detection the answer comes from the recorded work session;
// otherwise it falls back to the fixed working-hours schedule, which an
// explicit start/stop still overrides.
func IsActive(cfg config.Config, s state.State, t time.Time) bool {
	if cfg.AutoSessionDetect {
		return s.IsSessionActive()
	}
	if s.SessionState == state.SessionStateEnded && s.SessionManual {
		return false
	}
	if s.IsSessionActive() && s.SessionManual {
		return true
	}
	return IsWorkingTime(cfg, t)
}
