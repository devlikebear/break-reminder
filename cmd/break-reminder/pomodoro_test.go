package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestPomodoroReportsPartialSaveFailure(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // This test asserts the English diagnostic contract.
	path := setupSessionTest(t, config.Default(), time.Unix(1700000000, 0))
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"pomodoro", "on"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "configuration saved") {
		t.Fatalf("want partial save error, got %v", err)
	}
	saved, err := config.Load()
	if err != nil || !saved.PomodoroEnabled() {
		t.Fatalf("saved mode missing: %v", err)
	}
}

func TestPomodoroRejectsExplicitNonPositiveValues(t *testing.T) {
	for _, flag := range []string{"work", "break", "long-break", "every"} {
		for _, value := range []string{"0", "-1"} {
			t.Run(flag+value, func(t *testing.T) {
				setupSessionTest(t, config.Default(), time.Unix(1700000000, 0))
				cmd := newRootCmd()
				cmd.SetArgs([]string{"pomodoro", "on", "--" + flag, value})
				if err := cmd.Execute(); err == nil {
					t.Fatal("expected validation failure")
				}
			})
		}
	}
}
