package breakscreen

import (
	"fmt"

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

	workMin := cfg.EffectiveWorkMin()
	breakMin := breakDurSec / 60

	switch cfg.BreakScreenMode {
	case "block":
		showOverlay(workMin, breakDurSec, breakStartUnix, todayWorkMin, todayBreakMin)
	case "notify":
		sendNotification(workMin, breakMin)
	case "ask":
		choice := askBreakMode()
		switch choice {
		case "block":
			cfg.BreakScreenMode = "block"
			if err := config.Save(cfg); err != nil {
				log.Warn().Err(err).Msg("Failed to save break_screen_mode preference")
			}
			showOverlay(workMin, breakDurSec, breakStartUnix, todayWorkMin, todayBreakMin)
		default:
			cfg.BreakScreenMode = "notify"
			if err := config.Save(cfg); err != nil {
				log.Warn().Err(err).Msg("Failed to save break_screen_mode preference")
			}
			sendNotification(workMin, breakMin)
		}
	default:
		sendNotification(workMin, breakMin)
	}
}

func sendNotification(workMin, breakMin int) {
	notifier := notify.NewNotifier()
	_ = notifier.Send("Break Time!", fmt.Sprintf("%d minutes complete! Take a %d-minute break~", workMin, breakMin), "Blow")
}
