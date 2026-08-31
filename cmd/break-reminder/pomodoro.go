package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newPomodoroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pomodoro",
		Short: "Switch between the classic and pomodoro timer",
		Long: "The pomodoro timer replaces the single long work block with short focus blocks " +
			"and a longer break after every few of them.",
	}

	var workMin, breakMin, longBreakMin, longBreakEvery int

	onCmd := &cobra.Command{
		Use:   "on",
		Short: "Enable pomodoro mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			changes := map[string]any{"timer_mode": config.TimerModePomodoro}
			if workMin > 0 {
				changes["pomodoro_work_min"] = workMin
			}
			if breakMin > 0 {
				changes["pomodoro_break_min"] = breakMin
			}
			if longBreakMin > 0 {
				changes["pomodoro_long_break_min"] = longBreakMin
			}
			if longBreakEvery > 0 {
				changes["pomodoro_long_break_every"] = longBreakEvery
			}

			updated, err := applyConfigChanges(changes)
			if err != nil {
				return err
			}
			resetCurrentCycle()

			fmt.Fprintf(cmd.OutOrStdout(), "Pomodoro mode on: %d min work / %d min break, %d min long break every %d pomodoros.\n",
				updated.PomodoroWorkMin, updated.PomodoroBreakMin, updated.PomodoroLongBreakMin, updated.PomodoroLongBreakEvery)
			return nil
		},
	}
	onCmd.Flags().IntVar(&workMin, "work", 0, "Focus block length in minutes")
	onCmd.Flags().IntVar(&breakMin, "break", 0, "Short break length in minutes")
	onCmd.Flags().IntVar(&longBreakMin, "long-break", 0, "Long break length in minutes")
	onCmd.Flags().IntVar(&longBreakEvery, "every", 0, "Number of pomodoros before a long break")

	offCmd := &cobra.Command{
		Use:   "off",
		Short: "Return to the classic timer",
		RunE: func(cmd *cobra.Command, args []string) error {
			updated, err := applyConfigChanges(map[string]any{"timer_mode": config.TimerModeClassic})
			if err != nil {
				return err
			}
			resetCurrentCycle()

			fmt.Fprintf(cmd.OutOrStdout(), "Classic mode on: %d min work / %d min break.\n",
				updated.WorkDurationMin, updated.BreakDurationMin)
			return nil
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current timer mode and today's pomodoro count",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _ := state.Load(state.DefaultStatePath())
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Timer mode:", timerModeSummary(cfg))
			if cfg.PomodoroEnabled() {
				fmt.Fprintf(out, "Cycle: %d / %d\n", s.PomodoroCount, cfg.PomodoroLongBreakEvery)
				fmt.Fprintf(out, "Completed today: %d\n", s.TodayPomodoros)
				fmt.Fprintf(out, "Next break: %d min\n", cfg.EffectiveBreakMin(s.PomodoroCount+1))
			}
			return nil
		},
	}

	cmd.AddCommand(onCmd, offCmd, statusCmd)
	return cmd
}

// applyConfigChanges validates and persists a partial config update.
func applyConfigChanges(changes map[string]any) (config.Config, error) {
	data, err := yaml.Marshal(changes)
	if err != nil {
		return cfg, err
	}
	updated, err := config.ApplyYAMLChanges(cfg, data)
	if err != nil {
		return cfg, fmt.Errorf("invalid config change: %w", err)
	}
	if err := config.Save(updated); err != nil {
		return cfg, fmt.Errorf("save config: %w", err)
	}
	cfg = updated
	return updated, nil
}

// resetCurrentCycle restarts the running work block so a mode switch does not
// trigger an immediate break with the work time counted under the old mode.
func resetCurrentCycle() {
	_ = state.Update(state.DefaultStatePath(), func(s state.State) (state.State, error) {
		if s.Mode == "work" {
			s.WorkSeconds = 0
			s.SnoozeUntil = 0
			s.PomodoroCount = 0
			s.LastCheck = nowFunc().Unix()
		}
		return s, nil
	})
}

// timerModeSummary describes the active timer mode in one line.
func timerModeSummary(c config.Config) string {
	if c.PomodoroEnabled() {
		return fmt.Sprintf("pomodoro (%d/%d min, %d min long break every %d)",
			c.PomodoroWorkMin, c.PomodoroBreakMin, c.PomodoroLongBreakMin, c.PomodoroLongBreakEvery)
	}
	return fmt.Sprintf("classic (%d/%d min)", c.WorkDurationMin, c.BreakDurationMin)
}
