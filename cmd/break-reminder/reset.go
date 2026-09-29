package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: i18n.Text("Reset the timer"),
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()
			if err := state.Update(state.DefaultStatePath(), func(state.State) (state.State, error) {
				s := state.New()
				s.LastCheck = now.Unix()
				return s, nil
			}); err != nil {
				return err
			}
			logging.Log(logging.DefaultLogPath(), i18n.Text("Timer manually reset"))
			fmt.Println(i18n.Text("Timer has been reset."))
			return nil
		},
	}
}
