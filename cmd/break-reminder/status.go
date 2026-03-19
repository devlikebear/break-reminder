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

			fmt.Println("🐹 Break Reminder Status")
			fmt.Println("========================")
			fmt.Println("System:", launchd.Status())

			if schedule.IsWorkingTime(cfg, now) {
				fmt.Println("State:  Active (Within working hours)")
			} else {
				fmt.Println("State:  Inactive (Outside working hours)")
			}

			fmt.Println("------------------------")
			fmt.Println("Mode:", s.Mode)
			fmt.Printf("Session Work: %dmin / %dmin\n", s.WorkSeconds/60, cfg.WorkDurationMin)
			fmt.Printf("Daily Stats: Work %s / Break %s\n", fmtMin(s.TodayWorkSeconds/60), fmtMin(s.TodayBreakSeconds/60))
			fmt.Printf("Current idle: %dsec\n", idleSec)

			if s.Mode == "break" {
				breakElapsed := int(now.Unix() - s.BreakStart)
				fmt.Printf("Break elapsed: %dmin / %dmin\n", breakElapsed/60, cfg.BreakDurationMin)
			}

			return nil
		},
	}
}
