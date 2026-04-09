package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/state"
)

const defaultSnoozeMinutes = 5

var snoozeNow = time.Now

func newSnoozeCmd() *cobra.Command {
	var duration time.Duration

	cmd := &cobra.Command{
		Use:   "snooze",
		Short: "End the current break early and postpone the next one",
		Long:  "Snooze is only valid during an active break. It returns to work mode immediately and postpones the next break by the requested duration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSnooze(state.DefaultStatePath(), snoozeNow(), duration)
		},
	}
	cmd.Flags().DurationVar(&duration, "for", defaultSnoozeMinutes*time.Minute, "How long to postpone the next break (for example 5m or 10m)")
	allowInvalidConfig(cmd)
	return cmd
}

func runSnooze(statePath string, now time.Time, duration time.Duration) error {
	var updated state.State
	if err := state.Update(statePath, func(current state.State) (state.State, error) {
		next, err := current.SnoozeBreak(now, duration)
		if err != nil {
			return current, err
		}
		updated = next
		return next, nil
	}); err != nil {
		switch {
		case errors.Is(err, state.ErrBreakNotActive):
			return fmt.Errorf("cannot snooze: no active break")
		case errors.Is(err, state.ErrStatePaused):
			return fmt.Errorf("cannot snooze while paused")
		default:
			return err
		}
	}

	fmt.Printf("Break snoozed for %s. Back to work until %s.\n", duration.Round(time.Second), time.Unix(updated.SnoozeUntil, 0).Format(time.RFC3339))
	return nil
}
