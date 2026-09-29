package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	parent := &cobra.Command{Use: "time-tools", Short: "Independent countdown timers"}
	var directory string
	var jsonOutput bool
	parent.PersistentFlags().StringVar(&directory, "data-dir", timetools.DefaultDirectory(), "Directory for time tools only (isolated testing)")
	parent.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Print machine-readable result")
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
		fmt.Fprintf(cmd.OutOrStdout(), "Time tools worker: %t\n", r.Running(time.Now().UnixMilli()))
		if c := s.Countdown; c != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (%s)\n", c.Label, c.Phase, c.ID)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No countdown timer")
		}
		return nil
	}
	status := &cobra.Command{Use: "status", Short: "Read time tool state", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, err := timetools.NewStore(directory).Load()
		return report(cmd, s, err)
	}}
	allowInvalidConfig(status)
	parent.AddCommand(status)
	run := &cobra.Command{Use: "run", Short: "Run the background time tools worker", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		worker := timetools.Worker{Store: timetools.NewStore(directory), NotificationAvailable: notify.Available, Send: func(ctx context.Context, e timetools.Event) error {
			return notify.SendEvent(ctx, "타이머 완료", e.Label+" 타이머가 끝났어요", e.ID)
		}}
		return worker.Run(ctx)
	}}
	allowInvalidConfig(run)
	parent.AddCommand(run)
	timerCmd := &cobra.Command{Use: "timer", Short: "Start or control a countdown"}
	parent.AddCommand(timerCmd)
	for _, kind := range []string{"start", "pause", "resume", "cancel", "restart", "acknowledge", "notify-again"} {
		kind := kind
		var id, label, duration, replaceID string
		var revision uint64
		cmd := &cobra.Command{Use: kind, Short: kind + " a time tool", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			store := timetools.NewStore(directory)
			command := timetools.Command{Kind: kind, ID: id, Label: label, ReplaceID: replaceID}
			bad := func(code, message string) error {
				return report(cmd, timetools.Snapshot{}, &timetools.Error{Code: code, Message: message})
			}
			if kind == "start" {
				d, err := time.ParseDuration(duration)
				if err != nil || d < time.Second || d > 24*time.Hour || d%time.Second != 0 {
					return bad("invalid_input", "시간은 1초부터 24시간까지 초 단위로 입력해 주세요")
				}
				command.DurationMS = d.Milliseconds()
				command.ID, err = timetools.NewID()
				if err != nil {
					return report(cmd, timetools.Snapshot{}, err)
				}
			} else if id == "" {
				return bad("invalid_input", "--id 또는 --event-id가 필요합니다")
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
					return bad("worker_unavailable", "시간 도구 worker가 실행 중이 아닙니다. 'break-reminder service install' 또는 'service start'로 복구해 주세요")
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
		cmd.Flags().Uint64Var(&revision, "if-revision", 0, "Reject stale state revision")
		if kind == "start" {
			cmd.Flags().StringVar(&duration, "duration", "", "Duration such as 5m or 10s")
			cmd.Flags().StringVar(&label, "label", "", "Optional timer name")
			cmd.Flags().StringVar(&replaceID, "replace-id", "", "Explicitly replace this timer ID")
		} else if kind == "acknowledge" || kind == "notify-again" {
			cmd.Flags().StringVar(&id, "event-id", "", "Completion event ID")
		} else {
			cmd.Flags().StringVar(&id, "id", "", "Target timer ID")
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
