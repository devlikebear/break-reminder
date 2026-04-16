package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/breakscreen"
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
	now := nowFunc()
	statePath := state.DefaultStatePath()
	logPath := logging.DefaultLogPath()

	// Check working hours
	if !schedule.IsWorkingTime(cfg, now) {
		if err := state.Update(statePath, func(s state.State) (state.State, error) {
			if !s.Paused {
				s.LastCheck = now.Unix()
			}
			return s, nil
		}); err != nil {
			log.Warn().Err(err).Msg("Failed to update state outside working hours, resetting state")
			recovered := state.New()
			recovered.LastCheck = now.Unix()
			return state.Save(statePath, recovered)
		}
		return nil
	}

	detector := idle.NewDetector()
	idleSec := detector.IdleSeconds()

	var result timer.TickResult
	if err := state.Update(statePath, func(s state.State) (state.State, error) {
		result = timer.Tick(cfg, s, now, idleSec)
		return result.State, nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to update state, using fresh state")
		result = timer.Tick(cfg, state.New(), now, idleSec)
		if saveErr := state.Save(statePath, result.State); saveErr != nil {
			return saveErr
		}
	}

	if result.LogMsg != "" {
		logging.Log(logPath, result.LogMsg)
	}

	executeActions(result.Actions, result.State, result.DayEndSummary)

	logging.Rotate(logPath, cfg.MaxLogLines)
	return nil
}

func executeActions(actions []timer.Action, s state.State, daySummary *timer.DayEndSummary) {
	notifier := notify.NewNotifier()
	speaker := tts.NewSpeaker(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, tts.ResolveAPIKey(cfg))

	for _, a := range actions {
		switch a {
		case timer.ActionNotifyBreakTime:
			breakscreen.Show(cfg, cfg.BreakDurationSec(), s.BreakStart)
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
		case timer.ActionSaveDailyHistory:
			if daySummary != nil {
				_ = ai.AppendHistory(ai.DailySummary{
					Date:     daySummary.Date,
					WorkMin:  daySummary.WorkSeconds / 60,
					BreakMin: daySummary.BreakSeconds / 60,
				})
			}
		}
	}
}
