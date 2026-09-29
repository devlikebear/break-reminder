package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/dashboard"
)

func newBreakCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "break [activity]",
		Short: i18n.Text("Start a guided break activity"),
		Long:  i18n.Text("Start a guided break activity. Available activities:\n  eye      - 20-20-20 eye exercise (2 min)\n  stretch  - Guided stretching (5 min)\n  breathe  - Box breathing exercise (4 min)\n  walk     - Walking countdown timer (5 min)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// Show activity selection
				fmt.Println(i18n.Text("Available break activities:"))
				fmt.Println(i18n.Text("  eye      - 20-20-20 eye exercise (2 min)"))
				fmt.Println(i18n.Text("  stretch  - Guided stretching (5 min)"))
				fmt.Println(i18n.Text("  breathe  - Box breathing exercise (4 min)"))
				fmt.Println(i18n.Text("  walk     - Walking countdown timer (5 min)"))
				fmt.Println()
				fmt.Println(i18n.Text("Usage: break-reminder break <activity>"))
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
				return fmt.Errorf(i18n.Text("unknown activity: %s"), args[0])
			}

			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf(i18n.Text("break activity: %w"), err)
			}
			return nil
		},
	}

	return cmd
}
