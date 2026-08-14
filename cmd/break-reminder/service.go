package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/autoupdate"
	"github.com/devlikebear/break-reminder/internal/breakscreen"
	"github.com/devlikebear/break-reminder/internal/launchd"
)

var (
	serviceExecutablePath = os.Executable
	serviceFindMenuBar    = breakscreen.FindHelper
	serviceDetectHomebrew = autoupdate.DetectHomebrewInstall
	serviceInstallAgents  = launchd.Install
	serviceInstallUpdater = launchd.InstallUpdater
	serviceDisableUpdater = launchd.DisableUpdater
	serviceTimerStatus    = launchd.Status
	serviceMenuBarStatus  = launchd.MenuBarStatus
	serviceUpdaterStatus  = launchd.UpdaterStatus
)

func newServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage launchd service",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install as macOS LaunchAgent",
			RunE: func(cmd *cobra.Command, args []string) error {
				exe, err := serviceExecutablePath()
				if err != nil {
					return fmt.Errorf("resolve executable path: %w", err)
				}
				menuBarPath := serviceFindMenuBar("break-menubar")
				homebrewInstall, installedByHomebrew := serviceDetectHomebrew(exe)
				if installedByHomebrew {
					exe = homebrewInstall.BinaryPath
					menuBarPath = homebrewInstall.MenuBarPath
				}

				menuBarInstalled, err := serviceInstallAgents(exe, menuBarPath)
				if err != nil {
					return err
				}
				if installedByHomebrew {
					if err := serviceInstallUpdater(exe); err != nil {
						return err
					}
				} else if err := serviceDisableUpdater(); err != nil {
					return fmt.Errorf("disable Homebrew auto-update: %w", err)
				}

				out := cmd.OutOrStdout()
				fmt.Fprintln(out, "Successfully installed and loaded break-reminder agent!")
				fmt.Fprintln(out, "It will now run every minute in the background.")
				if menuBarInstalled {
					fmt.Fprintln(out, "Menu bar app auto-start is enabled and will stay running in the background.")
				} else {
					fmt.Fprintln(out, "Menu bar auto-start skipped because break-menubar helper was not found.")
				}
				if installedByHomebrew {
					fmt.Fprintln(out, "Homebrew auto-update is enabled and will check daily at 04:00.")
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall macOS LaunchAgent",
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := launchd.Uninstall(); err != nil {
					return err
				}
				fmt.Println("Successfully uninstalled break-reminder agent.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start the agent",
			RunE: func(cmd *cobra.Command, args []string) error {
				return launchd.Start()
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop the agent",
			RunE: func(cmd *cobra.Command, args []string) error {
				return launchd.Stop()
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show agent status",
			Run: func(cmd *cobra.Command, args []string) {
				out := cmd.OutOrStdout()
				fmt.Fprintln(out, "Timer:", serviceTimerStatus())
				fmt.Fprintln(out, "Menu Bar:", serviceMenuBarStatus())
				fmt.Fprintln(out, "Auto Update:", serviceUpdaterStatus())
			},
		},
	)

	return cmd
}
