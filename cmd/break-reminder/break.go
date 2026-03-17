package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/dashboard"
)

func newBreakCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "break [activity]",
		Short: "Start a guided break activity",
		Long: `Start a guided break activity. Available activities:
  eye      - 20-20-20 eye exercise (2 min)
  stretch  - Guided stretching (5 min)
  breathe  - Box breathing exercise (4 min)
  walk     - Walking countdown timer (5 min)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// Show activity selection
				fmt.Println("Available break activities:")
				fmt.Println("  eye      - 20-20-20 eye exercise (2 min)")
				fmt.Println("  stretch  - Guided stretching (5 min)")
				fmt.Println("  breathe  - Box breathing exercise (4 min)")
				fmt.Println("  walk     - Walking countdown timer (5 min)")
				fmt.Println()
				fmt.Println("Usage: break-reminder break <activity>")
				return nil
			}

			var model tea.Model
			switch args[0] {
			case "eye":
				model = dashboard.NewEyeActivity()
			case "stretch":
				model = dashboard.NewStretchActivity()
			case "breathe":
				model = dashboard.NewBreatheActivity()
			case "walk":
				model = dashboard.NewWalkActivity()
			default:
				return fmt.Errorf("unknown activity: %s", args[0])
			}

			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("break activity: %w", err)
			}
			return nil
		},
	}

	return cmd
}
