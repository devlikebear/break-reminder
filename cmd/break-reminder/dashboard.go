package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/breakscreen"
	"github.com/devlikebear/break-reminder/internal/dashboard"
)

func newDashboardCmd() *cobra.Command {
	var gui bool

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: i18n.Text("Interactive TUI dashboard (--gui for native macOS window)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if gui {
				return runGUIDashboard()
			}
			m := dashboard.New(cfg, version)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf(i18n.Text("dashboard: %w"), err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&gui, "gui", false, i18n.Text("Launch native macOS dashboard window"))
	return cmd
}

func runGUIDashboard() error {
	helperPath := breakscreen.FindHelper("break-dashboard")
	if helperPath == "" {
		return helperNotFoundError("break-dashboard")
	}

	log.Info().Str("helper", helperPath).Msg(i18n.Text("Launching GUI dashboard"))

	cmd := exec.Command(helperPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
