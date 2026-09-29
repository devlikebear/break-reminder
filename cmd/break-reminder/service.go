package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"encoding/json"
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
		Short: i18n.Text("Manage launchd service"),
	}

	var priorJSON string
	migrate := &cobra.Command{Use: "migrate-runtime", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		var prior launchd.RuntimeInstallation
		if priorJSON == "" {
			return fmt.Errorf("%s", i18n.Text("--prior runtime snapshot required"))
		}
		if err := json.Unmarshal([]byte(priorJSON), &prior); err != nil {
			return err
		}
		exe, err := serviceExecutablePath()
		if err != nil {
			return err
		}
		menu := serviceFindMenuBar("break-menubar")
		if install, ok := serviceDetectHomebrew(exe); ok {
			exe = install.BinaryPath
			menu = install.MenuBarPath
		}
		return launchd.MigrateRuntime(exe, menu, prior)
	}}
	migrate.Flags().StringVar(&priorJSON, "prior", "", i18n.Text("Captured runtime installation JSON"))
	cmd.AddCommand(migrate)
	cmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: i18n.Text("Install as macOS LaunchAgent"),
			RunE: func(cmd *cobra.Command, args []string) error {
				exe, err := serviceExecutablePath()
				if err != nil {
					return fmt.Errorf(i18n.Text("resolve executable path: %w"), err)
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
					return fmt.Errorf(i18n.Text("disable Homebrew auto-update: %w"), err)
				}

				out := cmd.OutOrStdout()
				fmt.Fprintln(out, i18n.Text("Successfully installed and loaded break-reminder agent!"))
				fmt.Fprintln(out, i18n.Text("Work reminders run every minute; the independent time tools worker stays running in the background."))
				if menuBarInstalled {
					fmt.Fprintln(out, i18n.Text("Menu bar app auto-start is enabled and will stay running in the background."))
				} else {
					fmt.Fprintln(out, i18n.Text("Menu bar auto-start skipped because break-menubar helper was not found."))
				}
				if installedByHomebrew {
					fmt.Fprintln(out, i18n.Text("Homebrew auto-update is enabled and will check daily at 04:00."))
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: i18n.Text("Uninstall macOS LaunchAgent"),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := launchd.Uninstall(); err != nil {
					return err
				}
				fmt.Println(i18n.Text("Successfully uninstalled break-reminder agent."))
				return nil
			},
		},
		&cobra.Command{
			Use:   "start",
			Short: i18n.Text("Start the agent"),
			RunE: func(cmd *cobra.Command, args []string) error {
				return launchd.Start()
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: i18n.Text("Stop the agent"),
			RunE: func(cmd *cobra.Command, args []string) error {
				return launchd.Stop()
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: i18n.Text("Show agent status"),
			Run: func(cmd *cobra.Command, args []string) {
				out := cmd.OutOrStdout()
				fmt.Fprintln(out, i18n.Text("Timer:"), i18n.Text(serviceTimerStatus()))
				fmt.Fprintln(out, i18n.Text("Time tools:"), i18n.Text(launchd.TimeToolsStatus()))
				fmt.Fprintln(out, i18n.Text("Menu Bar:"), i18n.Text(serviceMenuBarStatus()))
				fmt.Fprintln(out, i18n.Text("Auto Update:"), i18n.Text(serviceUpdaterStatus()))
			},
		},
	)

	for _, child := range cmd.Commands() {
		allowInvalidConfig(child)
	}
	return cmd
}
