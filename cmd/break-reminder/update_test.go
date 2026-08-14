package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/autoupdate"
)

func restoreUpdateDependencies(t *testing.T) {
	t.Helper()
	oldExecutablePath := updateExecutablePath
	oldDetectHomebrew := updateDetectHomebrew
	oldRunCommand := updateRunCommand
	oldRestartRuntime := updateRestartRuntime
	t.Cleanup(func() {
		updateExecutablePath = oldExecutablePath
		updateDetectHomebrew = oldDetectHomebrew
		updateRunCommand = oldRunCommand
		updateRestartRuntime = oldRestartRuntime
	})
}

func TestUpdateCommandRejectsNonHomebrewInstall(t *testing.T) {
	restoreUpdateDependencies(t)
	updateExecutablePath = func() (string, error) { return "/Users/test/.local/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{}, false
	}
	updateRunCommand = func(_ context.Context, _ string, _ ...string) (string, error) {
		t.Fatal("non-Homebrew update executed a command")
		return "", nil
	}

	cmd := newUpdateCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Homebrew") {
		t.Fatalf("update error = %v, want Homebrew guidance", err)
	}
}

func TestUpdateCommandReportsCurrentFormulaWithoutRestart(t *testing.T) {
	restoreUpdateDependencies(t)
	updateExecutablePath = func() (string, error) { return "/opt/homebrew/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}, true
	}
	calls := 0
	updateRunCommand = func(_ context.Context, _ string, _ ...string) (string, error) {
		calls++
		return "", nil
	}
	updateRestartRuntime = func() error {
		t.Fatal("current formula restarted runtime")
		return nil
	}

	out := new(bytes.Buffer)
	cmd := newUpdateCmd()
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("Homebrew command calls = %d, want 2", calls)
	}
	if !strings.Contains(out.String(), "up to date") {
		t.Fatalf("output = %q, want up-to-date message", out.String())
	}
}

func TestAutomaticUpdateStaysQuietWhenFormulaIsCurrent(t *testing.T) {
	restoreUpdateDependencies(t)
	updateExecutablePath = func() (string, error) { return "/opt/homebrew/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}, true
	}
	updateRunCommand = func(_ context.Context, _ string, _ ...string) (string, error) { return "", nil }
	updateRestartRuntime = func() error {
		t.Fatal("current formula restarted runtime")
		return nil
	}

	out := new(bytes.Buffer)
	cmd := newUpdateCmd()
	cmd.SetArgs([]string{"--automatic"})
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("automatic update error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("automatic current output = %q, want silence", out.String())
	}
}

func TestUpdateCommandRestartsRuntimeAfterUpgrade(t *testing.T) {
	restoreUpdateDependencies(t)
	updateExecutablePath = func() (string, error) { return "/opt/homebrew/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}, true
	}
	updateRunCommand = func(_ context.Context, _ string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "outdated" {
			return "break-reminder\n", nil
		}
		return "", nil
	}
	restarted := false
	updateRestartRuntime = func() error {
		restarted = true
		return nil
	}

	out := new(bytes.Buffer)
	cmd := newUpdateCmd()
	cmd.SetArgs([]string{"--automatic"})
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update error = %v", err)
	}
	if !restarted {
		t.Fatal("updated formula did not restart runtime agents")
	}
	if !strings.Contains(out.String(), "updated successfully") {
		t.Fatalf("output = %q, want success message", out.String())
	}
}

func TestUpdateCommandProvidesRecoveryWhenRestartFails(t *testing.T) {
	restoreUpdateDependencies(t)
	updateExecutablePath = func() (string, error) { return "/opt/homebrew/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}, true
	}
	updateRunCommand = func(_ context.Context, _ string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "outdated" {
			return "break-reminder\n", nil
		}
		return "", nil
	}
	updateRestartRuntime = func() error { return errors.New("launchctl denied") }

	cmd := newUpdateCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "break-reminder service install") {
		t.Fatalf("restart error = %v, want actionable service install recovery", err)
	}
}

func TestRootRegistersUpdateAsConfigIndependentCommand(t *testing.T) {
	root := newRootCmd()
	cmd, _, err := root.Find([]string{"update"})
	if err != nil || cmd == root {
		t.Fatalf("root.Find(update) = %v, %v", cmd, err)
	}
	if !commandAllowsInvalidConfig(cmd) {
		t.Fatal("update command requires a valid user config")
	}
}
