package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newPauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause the timer without losing progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			statePath := state.DefaultStatePath()
			pausedMode := "work"
			alreadyPaused := false
			if err := state.Update(statePath, func(s state.State) (state.State, error) {
				if s.Paused {
					alreadyPaused = true
					pausedMode = s.Mode
					return s, nil
				}
				pausedMode = s.Mode
				return s.Pause(nowFunc().Unix()), nil
			}); err != nil {
				return err
			}
			if alreadyPaused {
				fmt.Fprintln(cmd.OutOrStdout(), "Timer is already paused.")
				return nil
			}

			logging.Log(logging.DefaultLogPath(), "Timer paused")
			fmt.Fprintf(cmd.OutOrStdout(), "Timer paused (%s mode).\n", pausedMode)
			return nil
		},
	}
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
