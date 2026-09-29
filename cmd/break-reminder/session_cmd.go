package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timer"
)

// sessionConfig loads the config but keeps working with defaults when the file
// is broken, so the timer can always be started or stopped by hand.
func sessionConfig(errOut io.Writer) config.Config {
	loaded, err := loadAppConfig()
	if err != nil {
		fmt.Fprintf(errOut, i18n.Text("Warning: config is invalid, using defaults (%v)\n"), err)
		return config.Default()
	}
	return loaded
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"clock-in"},
		Short:   i18n.Text("Start a work session now (manual clock-in)"),
		Long: i18n.Text("Starts the work/break cycle immediately — even outside working hours or on a day off — ") +
			i18n.Text("and clears any pause. The session runs until `break-reminder stop`, until you are away for ") +
			i18n.Text("session_idle_end_min, or until the day rolls over."),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCfg := sessionConfig(cmd.ErrOrStderr())
			now := nowFunc()
			statePath := state.DefaultStatePath()

			alreadyRunning := false
			var startedAt int64
			if err := state.Update(statePath, func(s state.State) (state.State, error) {
				if s.IsSessionActive() {
					alreadyRunning = true
					startedAt = s.SessionStart
					return s, nil
				}
				startedAt = now.Unix()
				return s.StartSession(now.Unix(), true), nil
			}); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if alreadyRunning {
				fmt.Fprintf(out, i18n.Text("Work session is already running (started at %s).\n"), formatClock(startedAt))
				return nil
			}

			logging.Log(logging.DefaultLogPath(), i18n.Text("Work session started (manual)"))
			fmt.Fprintf(out, i18n.Text("Work session started at %s. First break in %d minutes.\n"),
				formatClock(startedAt), appCfg.EffectiveWorkMin())
			return nil
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop",
		Aliases: []string{"clock-out"},
		Short:   i18n.Text("Stop the work session and hold reminders until the next start"),
		Long: i18n.Text("Ends the current work session and turns reminders off. Automatic detection stays off ") +
			i18n.Text("until the next day or until `break-reminder start`. Use `service stop` to unload the ") +
			i18n.Text("background agent entirely."),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionConfig(cmd.ErrOrStderr())
			now := nowFunc()
			statePath := state.DefaultStatePath()

			wasRunning := false
			var summary timer.SessionSummary
			if err := state.Update(statePath, func(s state.State) (state.State, error) {
				wasRunning = s.IsSessionActive()
				summary = timer.EndSessionSummary(s, now.Unix(), timer.SessionEndManual)
				return s.EndSession(now.Unix(), true), nil
			}); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if wasRunning {
				logging.Log(logging.DefaultLogPath(), i18n.Text("Work session stopped (manual)"))
				fmt.Fprintf(out, i18n.Text("Work session stopped at %s.\n"), formatClock(now.Unix()))
				if d := summary.Duration(); d > 0 {
					fmt.Fprintf(out, i18n.Text("Session length: %s\n"), fmtMin(d/60))
				}
			} else {
				logging.Log(logging.DefaultLogPath(), i18n.Text("Reminders stopped (manual, no active session)"))
				fmt.Fprintln(out, i18n.Text("No work session was running. Reminders are off until the next start."))
			}
			fmt.Fprintln(out, sessionEndMessage(&summary))
			fmt.Fprintln(out, i18n.Text("Reminders stay off until tomorrow or `break-reminder start`."))
			return nil
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}

func formatClock(unix int64) string {
	if unix <= 0 {
		return "unknown"
	}
	return time.Unix(unix, 0).Format("15:04")
}
