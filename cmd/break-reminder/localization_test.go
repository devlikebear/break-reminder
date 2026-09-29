package main

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"regexp"
	"strings"
	"testing"
)

func TestHelpUsesSelectedLanguageThroughoutCommandTree(t *testing.T) {
	for _, lang := range []string{"ko", "en"} {
		t.Run(lang, func(t *testing.T) {
			t.Setenv("BREAK_REMINDER_LANGUAGE", lang)
			for _, args := range [][]string{{"--help"}, {"time-tools", "timer", "start", "--help"}, {"config", "--help"}, {"completion", "--help"}, {"time-tools", "timer", "pause", "--help"}} {
				root := newRootCmd()
				var output bytes.Buffer
				root.SetOut(&output)
				root.SetErr(&output)
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				heading := "Usage:"
				if lang == "ko" {
					heading = "사용법:"
				}
				if !strings.Contains(output.String(), heading) {
					t.Fatalf("%v: %s", args, output.String())
				}
				if lang == "ko" && (strings.Contains(output.String(), "Available Commands:") || strings.Contains(output.String(), "help for") || strings.Contains(output.String(), "Global ") || strings.Contains(output.String(), "(default ")) {
					t.Fatal(output.String())
				}
				if !strings.Contains(output.String(), "--help") {
					t.Fatal("flag identifiers must be preserved")
				}
			}
		})
	}
}
func TestPauseKoreanOutputPreservesStoredIdentifiers(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "ko")
	t.Setenv("HOME", t.TempDir())
	root := newRootCmd()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"pause", "--mode", "focus"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "집중") || strings.Contains(output.String(), "focus") {
		t.Fatal(output.String())
	}
}

func TestAllCommandDescriptionsAndFlagHelpUseKorean(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "ko")
	var visit func(*cobra.Command)
	hasKorean := regexp.MustCompile(`[가-힣]`)
	visit = func(cmd *cobra.Command) {
		if cmd.Short != "" && !hasKorean.MatchString(cmd.Short) {
			t.Errorf("%s: %s", cmd.CommandPath(), cmd.Short)
		}
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if !hasKorean.MatchString(flag.Usage) {
				t.Errorf("%s --%s: %s", cmd.CommandPath(), flag.Name, flag.Usage)
			}
		})
		for _, child := range cmd.Commands() {
			visit(child)
		}
	}
	visit(newRootCmd())
}
