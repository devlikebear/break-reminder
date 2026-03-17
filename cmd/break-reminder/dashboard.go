package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/dashboard"
)

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Interactive TUI dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := dashboard.New(cfg)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("dashboard: %w", err)
			}
			return nil
		},
	}
}
