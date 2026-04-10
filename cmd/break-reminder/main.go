package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
)

var (
	version    = "dev"
	cfg        config.Config
	loadConfig = config.Load
	nowFunc    = time.Now
)

const allowInvalidConfigAnnotation = "allow-invalid-config"

func loadAppConfig() (config.Config, error) {
	return loadConfig()
}

func setAppConfig() error {
	loadedCfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	cfg = loadedCfg
	return nil
}

func allowInvalidConfig(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[allowInvalidConfigAnnotation] = "true"
}

func commandAllowsInvalidConfig(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Annotations[allowInvalidConfigAnnotation] == "true"
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "break-reminder",
		Short: "Smart work/break cycle enforcer for macOS",
		Long:  "Break Reminder - Work 50 minutes, rest 10 minutes, repeat!",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if commandAllowsInvalidConfig(cmd) {
				return nil
			}
			return setAppConfig()
		},
		SilenceUsage: true,
	}

	root.AddCommand(
		newCheckCmd(),
		newStatusCmd(),
		newDashboardCmd(),
		newMenuBarCmd(),
		newDaemonCmd(),
		newResetCmd(),
		newDoctorCmd(),
		newServiceCmd(),
		newBreakCmd(),
		newSnoozeCmd(),
		newConfigCmd(),
		newTTSCmd(),
		newAICmd(),
		newVersionCmd(),
	)

	return root
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})

	root := newRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("break-reminder", version)
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}
