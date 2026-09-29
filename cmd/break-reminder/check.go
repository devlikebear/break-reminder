package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/breakscreen"
	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/insights"
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
		Short: i18n.Text("Run a single timer check (used by launchd)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck()
		},
	}
}

func runCheck() error {
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

	executeActions(result.Actions, result.State, result.DayEndSummary, result.SessionEnded)

	logging.Rotate(logPath, cfg.MaxLogLines)
	return nil
}

// timerNeedsTick reports whether a full tick is worth running right now. A
// running session or an active pause always ticks — the session may need to be
// closed and the pause may be due to auto-resume — otherwise the tick only
// matters inside the session detection window (or the fixed working hours when
// automatic detection is off).
func timerNeedsTick(cfg config.Config, s state.State, now time.Time) bool {
	if s.IsSessionActive() || s.Paused {
		return true
	}
	// A stop made today holds until an explicit start; a stop made earlier still
	// needs a tick so the daily reset can release it.
	if s.SessionState == state.SessionStateEnded && s.SessionManual &&
		s.LastUpdateDate == now.Format("2006-01-02") {
		return false
	}
	if !schedule.IsWorkDay(cfg, now) {
		return false
	}
	if cfg.AutoSessionDetect {
		return schedule.InDetectWindow(cfg, now)
	}
	return schedule.IsWorkingTime(cfg, now)
}

func executeActions(actions []timer.Action, s state.State, daySummary *timer.DayEndSummary, sessionEnd *timer.SessionSummary) {
	notifier := notify.NewNotifier()
	speaker := tts.NewSpeaker(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, tts.ResolveAPIKey(cfg))
	workMin := cfg.EffectiveWorkMin()

	for _, a := range actions {
		switch a {
		case timer.ActionNotifyBreakTime:
			breakscreen.Show(cfg, cfg.EffectiveBreakSec(s.PomodoroCount), s.BreakStart)
		case timer.ActionNotifyBreakOver:
			_ = notifier.Send(i18n.Text("Break Over!"), fmt.Sprintf(i18n.Text("Back to work! %d-minute timer started~"), workMin), "Hero")
		case timer.ActionNotifyFiveMinWarning:
			_ = notifier.Send(i18n.Text("5 minutes left"), i18n.Text("Break time coming up~"), "")
		case timer.ActionNotifyStillOnBreak:
			_ = notifier.Send(i18n.Text("Still on break!"), i18n.Text("Keep resting!"), "")
		case timer.ActionNotifySessionStart:
			_ = notifier.Send(i18n.Text("Work session started"), fmt.Sprintf(i18n.Text("Timer is running — first break in %d minutes."), workMin), "Submarine")
		case timer.ActionNotifySessionEnd:
			_ = notifier.Send(i18n.Text("Work session ended"), sessionEndMessage(sessionEnd), "Glass")
		case timer.ActionSpeakBreakTime:
			if err := speaker.Speak(cfg.Voice, fmt.Sprintf(i18n.Text("Time for a break! You've been working for %d minutes."), workMin)); err != nil {
				log.Warn().Err(err).Msg(i18n.Text("TTS speak failed (break time)"))
			}
		case timer.ActionSpeakBreakOver:
			if err := speaker.Speak(cfg.Voice, i18n.Text("Break time is over! Let's get back to work!")); err != nil {
				log.Warn().Err(err).Msg(i18n.Text("TTS speak failed (break over)"))
			}
		case timer.ActionSaveDailyHistory:
			if daySummary != nil {
				var hourlyMin [24]int
				for i, s := range daySummary.HourlyWork {
					hourlyMin[i] = s / 60
				}
				_ = ai.AppendHistory(ai.DailySummary{
					Date:       daySummary.Date,
					WorkMin:    daySummary.WorkSeconds / 60,
					BreakMin:   daySummary.BreakSeconds / 60,
					HourlyWork: hourlyMin,
				})

				if cfg.AIEnabled {
					go generateDailyInsights()
				}
			}
		}
	}
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
func sessionEndMessage(sum *timer.SessionSummary) string {
	if sum == nil {
		return i18n.Text("Timer stopped. See you next session!")
	}

	parts := []string{
		fmt.Sprintf(i18n.Text("Work %s"), fmtMin(sum.WorkSeconds/60)),
		fmt.Sprintf(i18n.Text("Break %s"), fmtMin(sum.BreakSeconds/60)),
	}
	if sum.Pomodoros > 0 {
		parts = append(parts, fmt.Sprintf(i18n.Text("%d pomodoro%s"), sum.Pomodoros, pluralS(sum.Pomodoros)))
	}
	return i18n.Text("Today · ") + strings.Join(parts, " · ")
}
