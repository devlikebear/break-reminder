package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/breakscreen"
	"github.com/devlikebear/break-reminder/internal/enginelock"
	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/insights"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/notify"
	"github.com/devlikebear/break-reminder/internal/paths"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timer"
	"github.com/devlikebear/break-reminder/internal/tts"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: i18n.Text("Run a single timer check (used by launchd)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck()
		},
	}
}

func runCheck() error {
	// A running engine (menu bar app) owns the timer state; stand aside.
	if enginelock.Held(paths.EngineLockFile()) {
		return nil
	}

	now := nowFunc()
	statePath := state.DefaultStatePath()
	logPath := logging.DefaultLogPath()

	// Skip idle detection and the full tick while nothing can happen;
	// timer.Tick owns every scheduling decision from here on.
	if current, err := state.Load(statePath); err == nil && !timerNeedsTick(cfg, current, now) {
		if err := state.Update(statePath, func(s state.State) (state.State, error) {
			if !s.Paused {
				s.LastCheck = now.Unix()
			}
			return s, nil
		}); err != nil {
			log.Warn().Err(err).Msg(i18n.Text("Failed to update state while the timer is off, resetting state"))
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
		log.Warn().Err(err).Msg(i18n.Text("Failed to update state, using fresh state"))
		result = timer.Tick(cfg, state.New(), now, idleSec)
		if saveErr := state.Save(statePath, result.State); saveErr != nil {
			return saveErr
		}
	}

	if result.LogMsg != "" {
		logging.Log(logPath, result.LogMsg)
	}

	executeActions(result)

	logging.Rotate(logPath, cfg.MaxLogLines)
	return nil
}

// timerNeedsTick is kept as a seam for tests; the logic lives in timer.
var timerNeedsTick = timer.NeedsTick

func executeActions(result timer.TickResult) {
	notifier := notify.NewNotifier()
	speaker := tts.NewSpeaker(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, tts.ResolveAPIKey(cfg))

	for _, e := range reminder.Events(cfg, result) {
		switch e.Type {
		case reminder.TypeBreakStart:
			breakscreen.Show(cfg, e.BreakSec, e.BreakStart)
		case reminder.TypeSpeak:
			if err := speaker.Speak(e.Voice, e.Text); err != nil {
				log.Warn().Err(err).Msg(i18n.Text("TTS speak failed"))
			}
		default:
			if e.Title != "" {
				_ = notifier.Send(e.Title, e.Body, e.Sound)
			}
		}
	}

	if result.DayEndSummary != nil {
		saveDailyHistory(*result.DayEndSummary)
		if cfg.AIEnabled {
			go generateDailyInsights()
		}
	}
}

// saveDailyHistory appends the finished day to the AI history file.
func saveDailyHistory(day timer.DayEndSummary) {
	var hourlyMin [24]int
	for i, s := range day.HourlyWork {
		hourlyMin[i] = s / 60
	}
	_ = ai.AppendHistory(ai.DailySummary{
		Date:       day.Date,
		WorkMin:    day.WorkSeconds / 60,
		BreakMin:   day.BreakSeconds / 60,
		HourlyWork: hourlyMin,
	})
}

func generateDailyInsights() {
	client := ai.NewClient(cfg.AICLI)
	if !client.Available() {
		log.Warn().Str("cli", cfg.AICLI).Msg(i18n.Text("AI CLI unavailable, skipping insights generation"))
		return
	}

	history, err := ai.LoadHistory()
	if err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Load history for insights"))
		return
	}
	recent := history
	if len(recent) > 7 {
		recent = recent[len(recent)-7:]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	report, err := insights.Generate(ctx, client, recent, time.Now())
	if err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Generate insights"))
		return
	}
	if err := insights.Save(report); err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Save insights"))
		return
	}
	log.Info().Msg(i18n.Text("Insights auto-generated"))
}

// sessionEndMessage renders the notification body for a finished work session.
func sessionEndMessage(sum *timer.SessionSummary) string { return reminder.SessionEndMessage(sum) }
