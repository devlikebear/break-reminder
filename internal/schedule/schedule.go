package schedule

import (
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

// IsWorkingTime checks if the given time falls within configured working hours and days.
func IsWorkingTime(cfg config.Config, t time.Time) bool {
	// Check day: time.Weekday() returns 0=Sun..6=Sat
	// Config uses ISO: 1=Mon..7=Sun
	isoDay := int(t.Weekday())
	if isoDay == 0 {
		isoDay = 7
	}

	dayMatch := false
	for _, d := range cfg.WorkDays {
		if d == isoDay {
			dayMatch = true
			break
		}
	}
	if !dayMatch {
		return false
	}

	hour := t.Hour()
	return hour >= cfg.WorkStartHour && hour < cfg.WorkEndHour
}
