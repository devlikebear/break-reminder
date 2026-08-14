package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/autoupdate"
	"github.com/devlikebear/break-reminder/internal/launchd"
)

const updateTimeout = 45 * time.Minute

var (
	updateExecutablePath = os.Executable
	updateDetectHomebrew = autoupdate.DetectHomebrewInstall
	updateRunCommand     = autoupdate.ExecuteCommand
	updateRestartRuntime = launchd.RestartRuntime
)

func newUpdateCmd() *cobra.Command {
	var automatic bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install a Homebrew update",
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := updateExecutablePath()
			if err != nil {
				return fmt.Errorf("resolve executable path: %w", err)
			}
			install, ok := updateDetectHomebrew(exe)
			if !ok {
				return fmt.Errorf("Homebrew update is unavailable: break-reminder was not installed by Homebrew")
			}

			ctx, cancel := updateContext(cmd.Context())
			defer cancel()
			result, err := autoupdate.CheckAndUpgrade(ctx, install, updateRunCommand)
			if err != nil {
				return err
			}
			if !result.Updated {
				if !automatic {
					fmt.Fprintln(cmd.OutOrStdout(), "break-reminder is already up to date.")
				}
				return nil
			}
			if err := updateRestartRuntime(); err != nil {
				return fmt.Errorf("restart services after update: %w; update was installed, run 'break-reminder service install' to recover", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "break-reminder updated successfully; services restarted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&automatic, "automatic", false, "run from the scheduled updater")
	_ = cmd.Flags().MarkHidden("automatic")
	allowInvalidConfig(cmd)
	return cmd
}

func updateContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, updateTimeout)
}
