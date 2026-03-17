package main

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/notify"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timer"
	"github.com/devlikebear/break-reminder/internal/tts"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Run a single timer check (used by launchd)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck()
		},
	}
}

func runCheck() error {
	now := time.Now()
	statePath := state.DefaultStatePath()
	logPath := logging.DefaultLogPath()

	// Check working hours
	if !schedule.IsWorkingTime(cfg, now) {
		s, _ := state.Load(statePath)
		s.LastCheck = now.Unix()
		return state.Save(statePath, s)
	}

	s, err := state.Load(statePath)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load state, using fresh state")
		s = state.New()
	}

	detector := idle.NewDetector()
	idleSec := detector.IdleSeconds()

	result := timer.Tick(cfg, s, now, idleSec)

	if result.LogMsg != "" {
		logging.Log(logPath, result.LogMsg)
	}

	executeActions(result.Actions)

	logging.Rotate(logPath, cfg.MaxLogLines)
	return state.Save(statePath, result.State)
}

func executeActions(actions []timer.Action) {
	notifier := notify.NewNotifier()
	speaker := tts.NewSpeaker()

	for _, a := range actions {
		switch a {
		case timer.ActionNotifyBreakTime:
			_ = notifier.Send("Break Time!", "50 minutes complete! Take a 10-minute break~", "Blow")
		case timer.ActionNotifyBreakOver:
			_ = notifier.Send("Break Over!", "Back to work! 50-minute timer started~", "Hero")
		case timer.ActionNotifyFiveMinWarning:
			_ = notifier.Send("5 minutes left", "Break time coming up~", "")
		case timer.ActionNotifyStillOnBreak:
			_ = notifier.Send("Still on break!", "Keep resting!", "")
		case timer.ActionSpeakBreakTime:
			_ = speaker.Speak(cfg.Voice, "Time for a break! You've been working for 50 minutes.")
		case timer.ActionSpeakBreakOver:
			_ = speaker.Speak(cfg.Voice, "Break time is over! Let's get back to work!")
		}
	}
}
