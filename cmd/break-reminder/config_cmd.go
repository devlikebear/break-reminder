package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: i18n.Text("Manage configuration"),
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: i18n.Text("Show current configuration"),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := yaml.Marshal(&cfg)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: i18n.Text("Open config in $EDITOR"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigFile(); err != nil {
				return fmt.Errorf(i18n.Text("ensure config: %w"), err)
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
	}
	allowInvalidConfig(editCmd)
	pathCmd := &cobra.Command{
		Use:   "path",
		Short: i18n.Text("Show config file path"),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.ConfigPath())
		},
	}
	allowInvalidConfig(pathCmd)

	setCmd := &cobra.Command{
		Use:   "set <key=value> [<key=value> ...]",
		Short: i18n.Text("Set one or more configuration values (validated, atomic write)"),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changes := map[string]any{}
			for _, arg := range args {
				k, v, ok := strings.Cut(arg, "=")
				if !ok || strings.TrimSpace(k) == "" {
					return fmt.Errorf(i18n.Text("invalid argument %q (expected key=value)"), arg)
				}
				parsed, err := parseConfigValue(strings.TrimSpace(v))
				if err != nil {
					return fmt.Errorf(i18n.Text("invalid value for %s: %w"), k, err)
				}
				changes[strings.TrimSpace(k)] = parsed
			}

			data, err := yaml.Marshal(changes)
			if err != nil {
				return err
			}

			updated, err := config.ApplyYAMLChanges(cfg, data)
			if err != nil {
				return fmt.Errorf(i18n.Text("invalid config change: %w"), err)
			}

			if err := config.Save(updated); err != nil {
				return fmt.Errorf(i18n.Text("save config: %w"), err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), i18n.Text("Configuration updated (%d key%s).\n"), len(changes), pluralS(len(changes)))
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: i18n.Text("Print a single config value"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := yaml.Marshal(&cfg)
			if err != nil {
				return err
			}
			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return err
			}
			val, ok := raw[args[0]]
			if !ok {
				return fmt.Errorf(i18n.Text("unknown config key %q"), args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}

	cmd.AddCommand(showCmd, editCmd, pathCmd, setCmd, getCmd)

	return cmd
}

func pluralS(n int) string {
	if n == 1 || i18n.Current() == "ko" {
		return ""
	}
	return "s"
}

// parseConfigValue converts a CLI-supplied raw string into a typed value
// suitable for YAML marshalling and subsequent config merge.
func parseConfigValue(v string) (any, error) {
	if v == "" {
		return "", nil
	}
	if v == "true" {
		return true, nil
	}
	if v == "false" {
		return false, nil
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i, nil
	}
	if strings.HasPrefix(v, "[") {
		var arr []any
		if err := yaml.Unmarshal([]byte(v), &arr); err != nil {
			return nil, fmt.Errorf(i18n.Text("malformed list: %w"), err)
		}
		return arr, nil
	}
	return v, nil
}
