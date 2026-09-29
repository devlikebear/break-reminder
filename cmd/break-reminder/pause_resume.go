package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

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
		Short: i18n.Text("Pause the timer without losing progress"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !state.IsValidPauseReason(modeFlag) {
				return fmt.Errorf(i18n.Text("invalid --mode %q (must be meeting|focus|afk)"), modeFlag)
			}
			var durationSec int
			if durationFlag != "" {
				d, err := time.ParseDuration(durationFlag)
				if err != nil {
					return fmt.Errorf(i18n.Text("invalid --duration %q: %w"), durationFlag, err)
				}
				if d <= 0 {
					return fmt.Errorf(i18n.Text("--duration must be positive, got %q"), durationFlag)
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
				fmt.Fprintln(cmd.OutOrStdout(), i18n.Text("Timer is already paused."))
				return nil
			}

			logging.Log(logging.DefaultLogPath(), fmt.Sprintf(i18n.Text("Timer paused (reason=%s, duration=%ds)"), modeFlag, durationSec))
			if durationSec > 0 {
				resumeAt := time.Unix(pauseAt+int64(durationSec), 0).Format("15:04")
				fmt.Fprintf(cmd.OutOrStdout(), i18n.Text("Timer paused (%s mode, reason=%s, auto-resume at %s).\n"), i18n.Text(pausedMode), i18n.Text(modeFlag), resumeAt)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), i18n.Text("Timer paused (%s mode, reason=%s).\n"), i18n.Text(pausedMode), i18n.Text(modeFlag))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&modeFlag, "mode", state.PauseReasonMeeting, i18n.Text("Pause mode: meeting|focus|afk"))
	cmd.Flags().StringVar(&durationFlag, "duration", "", i18n.Text("Auto-resume after duration (e.g., 30m, 1h). Empty = no auto-resume"))
	allowInvalidConfig(cmd)
	return cmd
}

func newResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: i18n.Text("Resume the timer from its paused mode"),
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
				fmt.Fprintln(cmd.OutOrStdout(), i18n.Text("Timer is not paused."))
				return nil
			}

			logging.Log(logging.DefaultLogPath(), i18n.Text("Timer resumed"))
			fmt.Fprintf(cmd.OutOrStdout(), i18n.Text("Timer resumed (%s mode).\n"), i18n.Text(mode))
			return nil
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}
