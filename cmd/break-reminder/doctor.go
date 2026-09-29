package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: i18n.Text("Diagnose and test all features"),
		Run: func(cmd *cobra.Command, args []string) {
			report := doctor.Run(cfg)

			green := "\033[0;32m"
			red := "\033[0;31m"
			yellow := "\033[1;33m"
			cyan := "\033[0;36m"
			nc := "\033[0m"

			fmt.Printf(i18n.Text("%s🩺 Break Reminder Doctor%s\n"), cyan, nc)
			fmt.Println("========================")
			fmt.Println()

			for _, c := range report.Checks {
				var color string
				var icon string
				switch c.Status {
				case "ok":
					color = green
					icon = "OK"
				case "warn":
					color = yellow
					icon = "WARN"
				case "info":
					color = yellow
					icon = "INFO"
				case "fail":
					color = red
					icon = "FAIL"
				}
				fmt.Printf("%-30s %s%s%s  %s\n", i18n.Text(c.Name), color, i18n.Text(icon), nc, i18n.Text(c.Detail))
			}

			fmt.Println()
			fmt.Println("========================")
			fails := report.FailCount()
			if fails == 0 {
				fmt.Printf(i18n.Text("Result: %sAll checks passed!%s (%d tests)\n"), green, nc, len(report.Checks))
			} else {
				fmt.Printf(i18n.Text("Result: %s%d failed%s, %d passed\n"), red, fails, nc, len(report.Checks)-fails)
			}
		},
	}
}
