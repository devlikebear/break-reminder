package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/launchd"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
)

func fmtMin(min int) string {
	if min >= 60 {
		h := min / 60
		m := min % 60
		if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", min)
}

func displayMode(s state.State) string {
	if s.Paused {
		return fmt.Sprintf("paused (%s)", s.Mode)
	}
	return s.Mode
}

// sessionStatusLine explains, in one line, whether the timer is running and why.
func sessionStatusLine(c config.Config, s state.State, now time.Time) string {
	if schedule.IsActive(c, s, now) {
		if s.IsSessionActive() && s.SessionStart > 0 {
			return fmt.Sprintf("Active (work session since %s)", formatClock(s.SessionStart))
		}
		return "Active (within working hours)"
	}
	if s.SessionState == state.SessionStateEnded && s.SessionManual {
		return "Stopped by hand (run `break-reminder start` to resume)"
	}
	if s.SessionState == state.SessionStateEnded && s.SessionEnd > 0 {
		return fmt.Sprintf("Session ended at %s (waiting for the next one)", formatClock(s.SessionEnd))
	}
	if c.AutoSessionDetect {
		if schedule.InDetectWindow(c, now) {
			return "Waiting (starts automatically on your next activity)"
		}
		return fmt.Sprintf("Inactive (auto-detect runs %02d:00-%02d:00 on working days)",
			c.SessionDetectStartHour, c.SessionDetectEndHour)
	}
	return "Inactive (outside working hours)"
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _ := state.Load(state.DefaultStatePath())
			detector := idle.NewDetector()
			idleSec := detector.IdleSeconds()
			now := nowFunc()
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "🐹 Break Reminder Status")
			fmt.Fprintln(out, "========================")
			fmt.Fprintln(out, "System:", launchd.Status())
			fmt.Fprintln(out, "Menu Bar:", launchd.MenuBarStatus())

			fmt.Fprintln(out, "Timer:", timerModeSummary(cfg))
			fmt.Fprintln(out, "State: ", sessionStatusLine(cfg, s, now))

			fmt.Fprintln(out, "------------------------")
			fmt.Fprintln(out, "Mode:", displayMode(s))
			fmt.Fprintf(out, "Session Work: %dmin / %dmin\n", s.WorkSeconds/60, cfg.EffectiveWorkMin())
			fmt.Fprintf(out, "Daily Stats: Work %s / Break %s\n", fmtMin(s.TodayWorkSeconds/60), fmtMin(s.TodayBreakSeconds/60))
			if cfg.PomodoroEnabled() {
				fmt.Fprintf(out, "Pomodoros: %d today (cycle %d/%d)\n", s.TodayPomodoros, s.PomodoroCount, cfg.PomodoroLongBreakEvery)
			}
			fmt.Fprintf(out, "Current idle: %dsec\n", idleSec)
			if s.Mode == "work" && s.SnoozeUntil > now.Unix() {
				fmt.Fprintf(out, "Next break postponed until: %s\n", time.Unix(s.SnoozeUntil, 0).Format(time.RFC3339))
			}
			if s.Paused && s.PausedAt > 0 {
				fmt.Fprintf(out, "Paused for: %s\n", now.Sub(time.Unix(s.PausedAt, 0)).Round(time.Second))
			}

			if s.Mode == "break" {
				referenceUnix := now.Unix()
				if s.Paused && s.PausedAt > 0 {
					referenceUnix = s.PausedAt
				}
				breakElapsed := int(referenceUnix - s.BreakStart)
				fmt.Fprintf(out, "Break elapsed: %dmin / %dmin\n", breakElapsed/60, cfg.EffectiveBreakMin(s.PomodoroCount))
			}

			return nil
		},
	}
}
