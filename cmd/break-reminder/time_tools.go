package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devlikebear/break-reminder/internal/notify"
	"github.com/devlikebear/break-reminder/internal/timetools"
	"github.com/spf13/cobra"
)

type timeToolsResponse struct {
	SchemaVersion int                `json:"schema_version"`
	Snapshot      timetools.Snapshot `json:"snapshot"`
	Runtime       timetools.Runtime  `json:"runtime"`
}

func newTimeToolsCmd() *cobra.Command {
	parent := &cobra.Command{Use: "time-tools", Short: i18n.Text("Independent countdown timers")}
	var directory string
	var jsonOutput bool
	parent.PersistentFlags().StringVar(&directory, "data-dir", timetools.DefaultDirectory(), i18n.Text("Directory for time tools only (isolated testing)"))
	parent.PersistentFlags().BoolVar(&jsonOutput, "json", false, i18n.Text("Print machine-readable result"))
	report := func(cmd *cobra.Command, s timetools.Snapshot, err error) error {
		if err != nil {
			if jsonOutput {
				_ = json.NewEncoder(cmd.OutOrStdout()).Encode(timetools.Error{Code: timetools.ErrorCode(err), Message: err.Error()})
			}
			return err
		}
		r := timetools.ReadRuntime(timetools.NewStore(directory))
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(timeToolsResponse{1, s, r})
		}
		fmt.Fprintf(cmd.OutOrStdout(), i18n.Text("Time tools worker: %t\n"), r.Running(time.Now().UnixMilli()))
		if c := s.Countdown; c != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (%s)\n", timeToolDisplayLabel(c.Label), i18n.Text(c.Phase), c.ID)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), i18n.Text("No countdown timer"))
		}
		return nil
	}
	status := &cobra.Command{Use: "status", Short: i18n.Text("Read time tool state"), Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, err := timetools.NewStore(directory).Load()
		return report(cmd, s, err)
	}}
	allowInvalidConfig(status)
	parent.AddCommand(status)
	run := &cobra.Command{Use: "run", Short: i18n.Text("Run the background time tools worker"), Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		worker := timetools.Worker{Store: timetools.NewStore(directory), NotificationAvailable: notify.Available, Send: func(ctx context.Context, e timetools.Event) error {
			return notify.SendEvent(ctx, i18n.Text("Timer complete"), i18n.Text("Timer complete: {0}", timeToolDisplayLabel(e.Label)), e.ID)
		}}
		return worker.Run(ctx)
	}}
	allowInvalidConfig(run)
	parent.AddCommand(run)
	timerCmd := &cobra.Command{Use: "timer", Short: i18n.Text("Start or control a countdown")}
	parent.AddCommand(timerCmd)
	for _, kind := range []string{"start", "pause", "resume", "cancel", "restart", "acknowledge", "notify-again"} {
		kind := kind
		var id, label, duration, replaceID string
		var revision uint64
		cmd := &cobra.Command{Use: kind, Short: i18n.Text(map[string]string{"pause": "Pause a time tool", "resume": "Resume a time tool", "cancel": "Cancel a time tool", "acknowledge": "Dismiss a time tool", "start": "Start a time tool", "restart": "Restart a time tool", "notify-again": "Repeat a time tool notification"}[kind]), Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			store := timetools.NewStore(directory)
			command := timetools.Command{Kind: kind, ID: id, Label: label, ReplaceID: replaceID}
			bad := func(code, message string) error {
				return report(cmd, timetools.Snapshot{}, &timetools.Error{Code: code, Message: message})
			}
			if kind == "start" {
				d, err := time.ParseDuration(duration)
				if err != nil || d < time.Second || d > 24*time.Hour || d%time.Second != 0 {
					return bad("invalid_input", i18n.Text("Enter a duration from 1 second to 24 hours in whole seconds"))
				}
				command.DurationMS = d.Milliseconds()
				command.ID, err = timetools.NewID()
				if err != nil {
					return report(cmd, timetools.Snapshot{}, err)
				}
			} else if id == "" {
				return bad("invalid_input", i18n.Text("--id or --event-id is required"))
			}
			if kind == "restart" {
				var err error
				command.NewID, err = timetools.NewID()
				if err != nil {
					return report(cmd, timetools.Snapshot{}, err)
				}
			}
			if kind == "start" || kind == "restart" || kind == "resume" || kind == "notify-again" {
				if !timetools.ReadRuntime(store).Running(time.Now().UnixMilli()) {
					return bad("worker_unavailable", i18n.Text("The time-tools worker is stopped. Run 'break-reminder service install' or 'service start' to recover"))
				}
			}
			var expected *uint64
			if cmd.Flags().Changed("if-revision") {
				expected = &revision
			}
			s, err := store.Update(expected, func(s timetools.Snapshot) (timetools.Snapshot, error) {
				return timetools.Apply(s, command, time.Now().UnixMilli())
			})
			return report(cmd, s, err)
		}}
		cmd.Flags().Uint64Var(&revision, "if-revision", 0, i18n.Text("Reject stale state revision"))
		if kind == "start" {
			cmd.Flags().StringVar(&duration, "duration", "", i18n.Text("Duration such as 5m or 10s"))
			cmd.Flags().StringVar(&label, "label", "", i18n.Text("Optional timer name"))
			cmd.Flags().StringVar(&replaceID, "replace-id", "", i18n.Text("Explicitly replace this timer ID"))
		} else if kind == "acknowledge" || kind == "notify-again" {
			cmd.Flags().StringVar(&id, "event-id", "", i18n.Text("Completion event ID"))
		} else {
			cmd.Flags().StringVar(&id, "id", "", i18n.Text("Target timer ID"))
		}
		allowInvalidConfig(cmd)
		if kind == "acknowledge" || kind == "notify-again" {
			parent.AddCommand(cmd)
		} else {
			timerCmd.AddCommand(cmd)
		}
	}
	return parent
}

func timeToolDisplayLabel(label string) string {
	if label == "" {
		return i18n.Text("Timer")
	}
	return label
}
