package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/logging"
)

func newDaemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon",
		Short: "Run as foreground daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logging.Log(logging.DefaultLogPath(), "Daemon started")
			fmt.Println("🐹 Break Reminder daemon started (Ctrl+C to stop)")

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			ticker := time.NewTicker(time.Duration(cfg.CheckIntervalSec) * time.Second)
			defer ticker.Stop()

			// Run immediately once
			if err := runCheck(); err != nil {
				log.Error().Err(err).Msg("Check failed")
			}

			for {
				select {
				case <-ticker.C:
					if newCfg, err := config.Load(); err == nil {
						cfg = newCfg
					} else {
						log.Warn().Err(err).Msg("Ignoring invalid config reload")
					}
					if err := runCheck(); err != nil {
						log.Error().Err(err).Msg("Check failed")
					}
				case <-sigCh:
					fmt.Println("\nDaemon stopped.")
					return nil
				}
			}
		},
	}
}
