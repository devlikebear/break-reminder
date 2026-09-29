package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	updateCaptureRuntime = launchd.CaptureRuntimeInstallation
	updateMigrateRuntime = func(ctx context.Context, binary string, prior launchd.RuntimeInstallation) error {
		data, err := json.Marshal(prior)
		if err != nil {
			return err
		}
		output, err := exec.CommandContext(ctx, binary, "service", "migrate-runtime", "--prior", string(data)).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		return nil
	}
)

func newUpdateCmd() *cobra.Command {
	var automatic bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: i18n.Text("Check for and install a Homebrew update"),
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := updateExecutablePath()
			if err != nil {
				return fmt.Errorf(i18n.Text("resolve executable path: %w"), err)
			}
			install, ok := updateDetectHomebrew(exe)
			if !ok {
				return fmt.Errorf("%s", i18n.Text("Homebrew update is unavailable: break-reminder was not installed by Homebrew"))
			}

			ctx, cancel := updateContext(cmd.Context())
			defer cancel()
			prior := updateCaptureRuntime()
			result, err := autoupdate.CheckAndUpgrade(ctx, install, updateRunCommand)
			if err != nil {
				return err
			}
			if !result.Updated {
				if !automatic {
					fmt.Fprintln(cmd.OutOrStdout(), i18n.Text("break-reminder is already up to date."))
				}
				return nil
			}
			if err := updateMigrateRuntime(ctx, install.BinaryPath, prior); err != nil {
				return fmt.Errorf(i18n.Text("restart services after update: %w; update was installed, run 'break-reminder service install' to recover"), err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), i18n.Text("break-reminder updated successfully; runtime services reconciled (stopped services remain stopped)."))
			return nil
		},
	}
	cmd.Flags().BoolVar(&automatic, "automatic", false, i18n.Text("run from the scheduled updater"))
	_ = cmd.Flags().MarkHidden("automatic")
	allowInvalidConfig(cmd)
	return cmd
}

func updateContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, updateTimeout)
}
