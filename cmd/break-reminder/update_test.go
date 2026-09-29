package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/autoupdate"
	"github.com/devlikebear/break-reminder/internal/launchd"
)

func restoreUpdateDependencies(t *testing.T) {
	t.Helper()
	oldCapture := updateCaptureRuntime
	updateCaptureRuntime = func() launchd.RuntimeInstallation { return launchd.RuntimeInstallation{} }
	oldExecutablePath := updateExecutablePath
	oldDetectHomebrew := updateDetectHomebrew
	oldRunCommand := updateRunCommand
	oldRestartRuntime := updateMigrateRuntime
	t.Cleanup(func() {
		updateCaptureRuntime = oldCapture
		updateExecutablePath = oldExecutablePath
		updateDetectHomebrew = oldDetectHomebrew
		updateRunCommand = oldRunCommand
		updateMigrateRuntime = oldRestartRuntime
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
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // This test asserts the English diagnostic contract.
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
	updateMigrateRuntime = func(_ context.Context, _ string, _ launchd.RuntimeInstallation) error {
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
	updateMigrateRuntime = func(_ context.Context, _ string, _ launchd.RuntimeInstallation) error {
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
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // This test asserts the English diagnostic contract.
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
	updateMigrateRuntime = func(_ context.Context, _ string, _ launchd.RuntimeInstallation) error {
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
	updateMigrateRuntime = func(_ context.Context, _ string, _ launchd.RuntimeInstallation) error {
		return errors.New("launchctl denied")
	}

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

func TestUpdateInvokesNewBinaryWithPriorRuntime(t *testing.T) {
	restoreUpdateDependencies(t)
	oldCapture := updateCaptureRuntime
	t.Cleanup(func() { updateCaptureRuntime = oldCapture })
	updateExecutablePath = func() (string, error) { return "/opt/homebrew/bin/break-reminder", nil }
	updateDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew", BinaryPath: "/opt/homebrew/bin/break-reminder"}, true
	}
	updateCaptureRuntime = func() launchd.RuntimeInstallation {
		return launchd.RuntimeInstallation{Timer: launchd.JobInstallation{Installed: true, Loaded: false}}
	}
	calls := []string{}
	updateRunCommand = func(_ context.Context, _ string, args ...string) (string, error) {
		calls = append(calls, args[0])
		if args[0] == "outdated" {
			return "break-reminder", nil
		}
		return "", nil
	}
	updateMigrateRuntime = func(_ context.Context, path string, prior launchd.RuntimeInstallation) error {
		if path != "/opt/homebrew/bin/break-reminder" || !prior.Timer.Installed || prior.Timer.Loaded {
			t.Fatal(path, prior)
		}
		calls = append(calls, "migration")
		return nil
	}
	cmd := newUpdateCmd()
	cmd.SetOut(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "update,outdated,upgrade,migration" {
		t.Fatal(calls)
	}
}
