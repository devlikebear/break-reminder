package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/breakscreen"
)

func newMenuBarCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "menubar",
		Short: "Launch the native macOS menu bar app",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMenuBarApp()
		},
	}
}

func runMenuBarApp() error {
	helperPath := breakscreen.FindHelper("break-menubar")
	if helperPath == "" {
		return fmt.Errorf("break-menubar helper not found. Run 'make build-helper' to build it")
	}

	log.Info().Str("helper", helperPath).Msg("Launching menu bar app")

	cmd := exec.Command(helperPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
