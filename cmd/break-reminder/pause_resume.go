package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newPauseCmd() *cobra.Command {
	var modeFlag string
	var durationFlag string
	cmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause the timer without losing progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !state.IsValidPauseReason(modeFlag) {
				return fmt.Errorf("invalid --mode %q (must be meeting|focus|afk)", modeFlag)
			}
			var durationSec int
			if durationFlag != "" {
				d, err := time.ParseDuration(durationFlag)
				if err != nil {
					return fmt.Errorf("invalid --duration %q: %w", durationFlag, err)
				}
				if d <= 0 {
					return fmt.Errorf("--duration must be positive, got %q", durationFlag)
				}
				durationSec = int(d.Seconds())
			}
			statePath := state.DefaultStatePath()
			pausedMode := "work"
			alreadyPaused := false
			pauseAt := nowFunc().Unix()
			if err := state.Update(statePath, func(s state.State) (state.State, error) {
				if s.Paused {
					alreadyPaused = true
					pausedMode = s.Mode
					return s, nil
				}
				pausedMode = s.Mode
				return s.Pause(pauseAt, modeFlag, durationSec), nil
			}); err != nil {
				return err
			}
			if alreadyPaused {
				fmt.Fprintln(cmd.OutOrStdout(), "Timer is already paused.")
				return nil
			}

			logging.Log(logging.DefaultLogPath(), fmt.Sprintf("Timer paused (reason=%s, duration=%ds)", modeFlag, durationSec))
			if durationSec > 0 {
				resumeAt := time.Unix(pauseAt+int64(durationSec), 0).Format("15:04")
				fmt.Fprintf(cmd.OutOrStdout(), "Timer paused (%s mode, reason=%s, auto-resume at %s).\n", pausedMode, modeFlag, resumeAt)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Timer paused (%s mode, reason=%s).\n", pausedMode, modeFlag)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&modeFlag, "mode", state.PauseReasonMeeting, "Pause mode: meeting|focus|afk")
	cmd.Flags().StringVar(&durationFlag, "duration", "", "Auto-resume after duration (e.g., 30m, 1h). Empty = no auto-resume")
	allowInvalidConfig(cmd)
	return cmd
}

func newResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume the timer from its paused mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			statePath := state.DefaultStatePath()
			mode := "work"
			notPaused := false
			if err := state.Update(statePath, func(s state.State) (state.State, error) {
				if !s.Paused {
					notPaused = true
					mode = s.Mode
					return s, nil
				}
				mode = s.Mode
				return s.Resume(nowFunc().Unix()), nil
			}); err != nil {
				return err
			}
			if notPaused {
				fmt.Fprintln(cmd.OutOrStdout(), "Timer is not paused.")
				return nil
			}

			logging.Log(logging.DefaultLogPath(), "Timer resumed")
			fmt.Fprintf(cmd.OutOrStdout(), "Timer resumed (%s mode).\n", mode)
			return nil
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}
