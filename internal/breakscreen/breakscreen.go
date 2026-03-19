package breakscreen

import (
	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/notify"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/rs/zerolog/log"
)

// Show handles the break screen display based on the configured mode.
// It may block until the break screen is dismissed (in "block" mode).
func Show(cfg config.Config, breakDurSec int, breakStartUnix int64) {
	// Load today's stats to pass to the overlay
	s, _ := state.Load(state.DefaultStatePath())
	todayWorkMin := s.TodayWorkSeconds / 60
	todayBreakMin := s.TodayBreakSeconds / 60

	switch cfg.BreakScreenMode {
	case "block":
		showOverlay(breakDurSec, breakStartUnix, todayWorkMin, todayBreakMin)
	case "notify":
		sendNotification()
	case "ask":
		choice := askBreakMode()
		switch choice {
		case "block":
			cfg.BreakScreenMode = "block"
			if err := config.Save(cfg); err != nil {
				log.Warn().Err(err).Msg("Failed to save break_screen_mode preference")
			}
			showOverlay(breakDurSec, breakStartUnix, todayWorkMin, todayBreakMin)
		default:
			cfg.BreakScreenMode = "notify"
			if err := config.Save(cfg); err != nil {
				log.Warn().Err(err).Msg("Failed to save break_screen_mode preference")
			}
			sendNotification()
		}
	default:
		sendNotification()
	}
}

func sendNotification() {
	notifier := notify.NewNotifier()
	_ = notifier.Send("Break Time!", "50 minutes complete! Take a 10-minute break~", "Blow")
}
