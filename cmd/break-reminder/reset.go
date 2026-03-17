package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset the timer",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := state.New()
			s.LastCheck = time.Now().Unix()
			if err := state.Save(state.DefaultStatePath(), s); err != nil {
				return err
			}
			logging.Log(logging.DefaultLogPath(), "Timer manually reset")
			fmt.Println("Timer has been reset.")
			return nil
		},
	}
}
