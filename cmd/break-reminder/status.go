package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

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

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _ := state.Load(state.DefaultStatePath())
			detector := idle.NewDetector()
			idleSec := detector.IdleSeconds()
			now := time.Now()
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "🐹 Break Reminder Status")
			fmt.Fprintln(out, "========================")
			fmt.Fprintln(out, "System:", launchd.Status())

			if schedule.IsWorkingTime(cfg, now) {
				fmt.Fprintln(out, "State:  Active (Within working hours)")
			} else {
				fmt.Fprintln(out, "State:  Inactive (Outside working hours)")
			}

			fmt.Fprintln(out, "------------------------")
			modeLabel := s.Mode
			if s.Paused {
				modeLabel = fmt.Sprintf("paused (%s)", s.Mode)
			}
			fmt.Fprintln(out, "Mode:", modeLabel)
			fmt.Fprintf(out, "Session Work: %dmin / %dmin\n", s.WorkSeconds/60, cfg.WorkDurationMin)
			fmt.Fprintf(out, "Daily Stats: Work %s / Break %s\n", fmtMin(s.TodayWorkSeconds/60), fmtMin(s.TodayBreakSeconds/60))
			fmt.Fprintf(out, "Current idle: %dsec\n", idleSec)
			if s.Mode == "work" && s.SnoozeUntil > now.Unix() {
				fmt.Fprintf(out, "Next break postponed until: %s\n", time.Unix(s.SnoozeUntil, 0).Format(time.RFC3339))
			}
			if s.Paused && s.PausedAt > 0 {
				fmt.Fprintf(out, "Paused for: %s\n", now.Sub(time.Unix(s.PausedAt, 0)).Round(time.Second))
			}

			if s.Mode == "break" {
				breakElapsed := int(now.Unix() - s.BreakStart)
				fmt.Fprintf(out, "Break elapsed: %dmin / %dmin\n", breakElapsed/60, cfg.BreakDurationMin)
			}

			return nil
		},
	}
}
