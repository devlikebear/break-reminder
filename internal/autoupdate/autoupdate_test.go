package autoupdate

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func stubHomebrewPaths(t *testing.T, missing string) {
	t.Helper()
	oldEvalSymlinks := evalSymlinks
	oldPathExists := pathExists
	evalSymlinks = func(path string) (string, error) {
		return "/opt/homebrew/Cellar/break-reminder/0.10.0/bin/break-reminder", nil
	}
	pathExists = func(path string) bool { return path != missing }
	t.Cleanup(func() {
		evalSymlinks = oldEvalSymlinks
		pathExists = oldPathExists
	})
}

func TestDetectHomebrewInstallReturnsStablePaths(t *testing.T) {
	stubHomebrewPaths(t, "")

	install, ok := DetectHomebrewInstall("/opt/homebrew/bin/break-reminder")
	if !ok {
		t.Fatal("DetectHomebrewInstall() did not recognize Homebrew Cellar path")
	}
	if got, want := install.BrewPath, "/opt/homebrew/bin/brew"; got != want {
		t.Fatalf("BrewPath = %q, want %q", got, want)
	}
	if got, want := install.BinaryPath, "/opt/homebrew/bin/break-reminder"; got != want {
		t.Fatalf("BinaryPath = %q, want %q", got, want)
	}
	if got, want := install.MenuBarPath, "/opt/homebrew/bin/break-menubar"; got != want {
		t.Fatalf("MenuBarPath = %q, want %q", got, want)
	}
}

func TestDetectHomebrewInstallRejectsMissingStableBrew(t *testing.T) {
	stubHomebrewPaths(t, "/opt/homebrew/bin/brew")

	if _, ok := DetectHomebrewInstall("/opt/homebrew/bin/break-reminder"); ok {
		t.Fatal("DetectHomebrewInstall() = true when stable brew path is missing")
	}
}

func TestDetectHomebrewInstallAllowsMissingOptionalMenuBar(t *testing.T) {
	stubHomebrewPaths(t, "/opt/homebrew/bin/break-menubar")

	install, ok := DetectHomebrewInstall("/opt/homebrew/bin/break-reminder")
	if !ok {
		t.Fatal("DetectHomebrewInstall() rejected Homebrew because optional menu bar helper is missing")
	}
	if install.MenuBarPath != "" {
		t.Fatalf("MenuBarPath = %q, want empty optional helper path", install.MenuBarPath)
	}
	if install.BinaryPath != "/opt/homebrew/bin/break-reminder" {
		t.Fatalf("BinaryPath = %q", install.BinaryPath)
	}
}

func TestDetectHomebrewInstallSupportsIntelPrefix(t *testing.T) {
	oldEvalSymlinks := evalSymlinks
	oldPathExists := pathExists
	evalSymlinks = func(path string) (string, error) {
		return "/usr/local/Cellar/break-reminder/0.10.0/bin/break-reminder", nil
	}
	pathExists = func(string) bool { return true }
	t.Cleanup(func() {
		evalSymlinks = oldEvalSymlinks
		pathExists = oldPathExists
	})

	install, ok := DetectHomebrewInstall("/usr/local/bin/break-reminder")
	if !ok {
		t.Fatal("DetectHomebrewInstall() rejected Intel Homebrew path")
	}
	if install.BrewPath != "/usr/local/bin/brew" || install.BinaryPath != "/usr/local/bin/break-reminder" {
		t.Fatalf("Intel stable paths = %+v", install)
	}
}

func TestDetectHomebrewInstallRejectsNonHomebrewExecutable(t *testing.T) {
	oldEvalSymlinks := evalSymlinks
	evalSymlinks = func(path string) (string, error) {
		return "/Users/test/.local/bin/break-reminder", nil
	}
	t.Cleanup(func() { evalSymlinks = oldEvalSymlinks })

	if _, ok := DetectHomebrewInstall("/Users/test/.local/bin/break-reminder"); ok {
		t.Fatal("DetectHomebrewInstall() recognized a non-Homebrew executable")
	}
}

func TestDetectHomebrewInstallRejectsUnresolvableExecutable(t *testing.T) {
	oldEvalSymlinks := evalSymlinks
	evalSymlinks = func(path string) (string, error) { return "", errors.New("missing") }
	t.Cleanup(func() { evalSymlinks = oldEvalSymlinks })

	if _, ok := DetectHomebrewInstall("/missing/break-reminder"); ok {
		t.Fatal("DetectHomebrewInstall() recognized an unresolvable executable")
	}
}

func TestCheckAndUpgradeStopsWhenFormulaIsCurrent(t *testing.T) {
	var calls [][]string
	run := func(_ context.Context, path string, args ...string) (string, error) {
		calls = append(calls, append([]string{path}, args...))
		return "", nil
	}
	install := HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}

	result, err := CheckAndUpgrade(context.Background(), install, run)
	if err != nil {
		t.Fatalf("CheckAndUpgrade() error = %v", err)
	}
	if result.Updated {
		t.Fatal("CheckAndUpgrade() reported an update for current formula")
	}
	want := [][]string{
		{"/opt/homebrew/bin/brew", "update"},
		{"/opt/homebrew/bin/brew", "outdated", "--formula", "--quiet", Formula},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("commands = %#v, want %#v", calls, want)
	}
}

func TestCheckAndUpgradeUpgradesOutdatedFormula(t *testing.T) {
	var calls [][]string
	run := func(_ context.Context, path string, args ...string) (string, error) {
		calls = append(calls, append([]string{path}, args...))
		if len(args) > 0 && args[0] == "outdated" {
			return "break-reminder\n", nil
		}
		return "", nil
	}
	install := HomebrewInstall{BrewPath: "/opt/homebrew/bin/brew"}

	result, err := CheckAndUpgrade(context.Background(), install, run)
	if err != nil {
		t.Fatalf("CheckAndUpgrade() error = %v", err)
	}
	if !result.Updated {
		t.Fatal("CheckAndUpgrade() did not report installed update")
	}
	wantLast := []string{"/opt/homebrew/bin/brew", "upgrade", Formula}
	if len(calls) != 3 || !reflect.DeepEqual(calls[2], wantLast) {
		t.Fatalf("commands = %#v, want third command %#v", calls, wantLast)
	}
}

type exitCodeError int

func (e exitCodeError) Error() string { return "exit status" }
func (e exitCodeError) ExitCode() int { return int(e) }

func TestCheckAndUpgradeAcceptsBrewOutdatedExitCodeOne(t *testing.T) {
	upgraded := false
	run := func(_ context.Context, _ string, args ...string) (string, error) {
		switch args[0] {
		case "outdated":
			return "devlikebear/tap/break-reminder\n", exitCodeError(1)
		case "upgrade":
			upgraded = true
		}
		return "", nil
	}

	result, err := CheckAndUpgrade(context.Background(), HomebrewInstall{BrewPath: "/brew"}, run)
	if err != nil {
		t.Fatalf("CheckAndUpgrade() error = %v", err)
	}
	if !result.Updated || !upgraded {
		t.Fatalf("result = %+v, upgraded = %v", result, upgraded)
	}
}

func TestCheckAndUpgradeRejectsExitOneWithoutFormulaOutput(t *testing.T) {
	upgraded := false
	run := func(_ context.Context, _ string, args ...string) (string, error) {
		if args[0] == "outdated" {
			return "network unavailable\n", exitCodeError(1)
		}
		if args[0] == "upgrade" {
			upgraded = true
		}
		return "", nil
	}

	_, err := CheckAndUpgrade(context.Background(), HomebrewInstall{BrewPath: "/brew"}, run)
	if err == nil || !strings.Contains(err.Error(), "brew outdated") {
		t.Fatalf("error = %v, want brew outdated failure", err)
	}
	if upgraded {
		t.Fatal("unexpected exit-one output triggered an upgrade")
	}
}

func TestCheckAndUpgradeRejectsUnexpectedSuccessfulOutput(t *testing.T) {
	upgraded := false
	run := func(_ context.Context, _ string, args ...string) (string, error) {
		if args[0] == "outdated" {
			return "unexpected output\n", nil
		}
		if args[0] == "upgrade" {
			upgraded = true
		}
		return "", nil
	}

	_, err := CheckAndUpgrade(context.Background(), HomebrewInstall{BrewPath: "/brew"}, run)
	if err == nil || !strings.Contains(err.Error(), "unexpected output") {
		t.Fatalf("error = %v, want unexpected-output failure", err)
	}
	if upgraded {
		t.Fatal("unexpected successful output triggered an upgrade")
	}
}

func TestCheckAndUpgradeAddsCommandContextToErrors(t *testing.T) {
	run := func(_ context.Context, path string, args ...string) (string, error) {
		return "", errors.New("network unavailable")
	}

	_, err := CheckAndUpgrade(context.Background(), HomebrewInstall{BrewPath: "/brew"}, run)
	if err == nil || !strings.Contains(err.Error(), "brew update") {
		t.Fatalf("error = %v, want brew update context", err)
	}
}
