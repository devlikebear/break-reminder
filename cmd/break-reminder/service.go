package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/breakscreen"
	"github.com/devlikebear/break-reminder/internal/launchd"
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
				exe, err := os.Executable()
				if err != nil {
					return fmt.Errorf("resolve executable path: %w", err)
				}
				menuBarPath := breakscreen.FindHelper("break-menubar")
				menuBarInstalled, err := launchd.Install(exe, menuBarPath)
				if err != nil {
					return err
				}
				fmt.Println("Successfully installed and loaded break-reminder agent!")
				fmt.Println("It will now run every minute in the background.")
				if menuBarInstalled {
					fmt.Println("Menu bar app auto-start is enabled and will stay running in the background.")
				} else {
					fmt.Println("Menu bar auto-start skipped because break-menubar helper was not found.")
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
				fmt.Println("Timer:", launchd.Status())
				fmt.Println("Menu Bar:", launchd.MenuBarStatus())
			},
		},
	)

	return cmd
}
