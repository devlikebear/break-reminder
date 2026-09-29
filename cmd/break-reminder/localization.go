package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Localize framework-owned help without changing command names, flags or values.
func localizeCommandHelp(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	cobra.AddTemplateFunc("localizeFlagDefaults", func(usage string) string {
		return strings.ReplaceAll(usage, "(default ", i18n.Text("(default "))
	})
	template := strings.ReplaceAll(root.UsageTemplate(), ".FlagUsages |", ".FlagUsages | localizeFlagDefaults |")
	for _, heading := range []string{"Usage:", "Aliases:", "Examples:", "Available Commands:", "Additional Commands:", "Global Flags:", "Flags:", "Additional help topics:"} {
		template = strings.ReplaceAll(template, heading, i18n.Text(heading))
	}
	template = strings.ReplaceAll(template, `Use "{{.CommandPath}} [command] --help" for more information about a command.`, i18n.Text("Use \"{0}\" for more information about a command.", "{{.CommandPath}} [command] --help"))
	root.SetUsageTemplate(template)
	var visit func(*cobra.Command)
	visit = func(cmd *cobra.Command) {
		cmd.InitDefaultHelpFlag()
		if help := cmd.Flags().Lookup("help"); help != nil {
			help.Usage = i18n.Text("Help for {0}", cmd.CommandPath())
		}
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { flag.Usage = i18n.Text(flag.Usage) })
		if cmd.Name() == "help" {
			cmd.Short = i18n.Text("Help for any command")
			cmd.Long = i18n.Text("Use help followed by a command name to see its options and usage.")
		}
		if cmd.Name() == "completion" {
			cmd.Short = i18n.Text("Generate shell completion scripts")
			cmd.Long = i18n.Text("Generate completion scripts for your shell. Choose a shell subcommand for setup instructions.")
		}
		if cmd.Parent() != nil && cmd.Parent().Name() == "completion" {
			shell := cmd.Name()
			cmd.Short = i18n.Text("Generate completion for {0}", shell)
			cmd.Long = i18n.Text("Generate the completion script and load it in your shell profile. Command:") + "\n\nbreak-reminder completion " + shell
		}
		originalArgs := cmd.Args
		if originalArgs != nil {
			cmd.Args = func(c *cobra.Command, args []string) error {
				if err := originalArgs(c, args); err != nil {
					return localizeCommandError(err)
				}
				return nil
			}
		}
		for _, child := range cmd.Commands() {
			visit(child)
		}
	}
	visit(root)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return localizeCommandError(err) })
}
func localizeCommandError(err error) error {
	if err == nil || i18n.Current() != "ko" {
		return err
	}
	message := err.Error()
	if parts := regexp.MustCompile(`^unknown command (".*") for (".*")$`).FindStringSubmatch(message); len(parts) == 3 {
		return errors.New(i18n.Text("Unknown command {0} for {1}", parts[1], parts[2]))
	}
	for _, prefix := range []string{"unknown flag: ", "unknown shorthand flag: ", "unknown command ", "flag needs an argument: "} {
		if strings.HasPrefix(message, prefix) {
			return errors.New(i18n.Text(prefix) + strings.TrimPrefix(message, prefix))
		}
	}
	var want, got int
	if _, e := fmt.Sscanf(message, "accepts %d arg(s), received %d", &want, &got); e == nil {
		return errors.New(i18n.Text("Expected {0} arguments, received {1}", want, got))
	}
	if _, e := fmt.Sscanf(message, "requires at least %d arg(s), only received %d", &want, &got); e == nil {
		return errors.New(i18n.Text("Expected at least {0} arguments, received {1}", want, got))
	}
	if _, e := fmt.Sscanf(message, "accepts at most %d arg(s), received %d", &want, &got); e == nil {
		return errors.New(i18n.Text("Expected at most {0} arguments, received {1}", want, got))
	}
	return err
}
