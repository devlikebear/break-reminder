package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose and test all features",
		Run: func(cmd *cobra.Command, args []string) {
			report := doctor.Run(cfg)

			green := "\033[0;32m"
			red := "\033[0;31m"
			yellow := "\033[1;33m"
			cyan := "\033[0;36m"
			nc := "\033[0m"

			fmt.Printf("%s🩺 Break Reminder Doctor%s\n", cyan, nc)
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
				fmt.Printf("%-30s %s%s%s  %s\n", c.Name, color, icon, nc, c.Detail)
			}

			fmt.Println()
			fmt.Println("========================")
			fails := report.FailCount()
			if fails == 0 {
				fmt.Printf("Result: %sAll checks passed!%s (%d tests)\n", green, nc, len(report.Checks))
			} else {
				fmt.Printf("Result: %s%d failed%s, %d passed\n", red, fails, nc, len(report.Checks)-fails)
			}
		},
	}
}
