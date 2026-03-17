package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show current configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				data, err := yaml.Marshal(&cfg)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			},
		},
		&cobra.Command{
			Use:   "edit",
			Short: "Open config in $EDITOR",
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := config.EnsureConfigFile(); err != nil {
					return fmt.Errorf("ensure config: %w", err)
				}

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}

				c := exec.Command(editor, config.ConfigPath())
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				return c.Run()
			},
		},
		&cobra.Command{
			Use:   "path",
			Short: "Show config file path",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(config.ConfigPath())
			},
		},
	)

	return cmd
}
