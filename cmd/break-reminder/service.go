package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

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
				if err := launchd.Install(exe); err != nil {
					return err
				}
				fmt.Println("Successfully installed and loaded break-reminder agent!")
				fmt.Println("It will now run every minute in the background.")
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
				fmt.Println(launchd.Status())
			},
		},
	)

	return cmd
}
