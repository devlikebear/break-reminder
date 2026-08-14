package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/autoupdate"
)

func TestServiceInstallEnablesUpdaterForHomebrew(t *testing.T) {
	oldExecutablePath := serviceExecutablePath
	oldFindMenuBar := serviceFindMenuBar
	oldDetectHomebrew := serviceDetectHomebrew
	oldInstallAgents := serviceInstallAgents
	oldInstallUpdater := serviceInstallUpdater
	oldDisableUpdater := serviceDisableUpdater
	t.Cleanup(func() {
		serviceExecutablePath = oldExecutablePath
		serviceFindMenuBar = oldFindMenuBar
		serviceDetectHomebrew = oldDetectHomebrew
		serviceInstallAgents = oldInstallAgents
		serviceInstallUpdater = oldInstallUpdater
		serviceDisableUpdater = oldDisableUpdater
	})

	serviceExecutablePath = func() (string, error) { return "/opt/homebrew/Cellar/break-reminder/0.10.0/bin/break-reminder", nil }
	serviceFindMenuBar = func(string) string { return "/opt/homebrew/Cellar/break-reminder/0.10.0/bin/break-menubar" }
	serviceDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{
			BrewPath:    "/opt/homebrew/bin/brew",
			BinaryPath:  "/opt/homebrew/bin/break-reminder",
			MenuBarPath: "/opt/homebrew/bin/break-menubar",
		}, true
	}
	var installedBinary, installedMenuBar, installedUpdater string
	serviceInstallAgents = func(binaryPath, menuBarPath string) (bool, error) {
		installedBinary, installedMenuBar = binaryPath, menuBarPath
		return true, nil
	}
	serviceInstallUpdater = func(binaryPath string) error {
		installedUpdater = binaryPath
		return nil
	}
	serviceDisableUpdater = func() error {
		t.Fatal("Homebrew install disabled updater")
		return nil
	}

	cmd := newServiceCmd()
	cmd.SetArgs([]string{"install"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("service install error = %v", err)
	}
	if installedBinary != "/opt/homebrew/bin/break-reminder" {
		t.Fatalf("timer binary = %q", installedBinary)
	}
	if installedMenuBar != "/opt/homebrew/bin/break-menubar" {
		t.Fatalf("menu bar binary = %q", installedMenuBar)
	}
	if installedUpdater != "/opt/homebrew/bin/break-reminder" {
		t.Fatalf("updater binary = %q", installedUpdater)
	}
}

func TestServiceInstallDisablesUpdaterOutsideHomebrew(t *testing.T) {
	oldExecutablePath := serviceExecutablePath
	oldFindMenuBar := serviceFindMenuBar
	oldDetectHomebrew := serviceDetectHomebrew
	oldInstallAgents := serviceInstallAgents
	oldInstallUpdater := serviceInstallUpdater
	oldDisableUpdater := serviceDisableUpdater
	t.Cleanup(func() {
		serviceExecutablePath = oldExecutablePath
		serviceFindMenuBar = oldFindMenuBar
		serviceDetectHomebrew = oldDetectHomebrew
		serviceInstallAgents = oldInstallAgents
		serviceInstallUpdater = oldInstallUpdater
		serviceDisableUpdater = oldDisableUpdater
	})

	serviceExecutablePath = func() (string, error) { return "/Users/test/.local/bin/break-reminder", nil }
	serviceFindMenuBar = func(string) string { return "/Users/test/.local/bin/break-menubar" }
	serviceDetectHomebrew = func(string) (autoupdate.HomebrewInstall, bool) {
		return autoupdate.HomebrewInstall{}, false
	}
	serviceInstallAgents = func(binaryPath, menuBarPath string) (bool, error) { return true, nil }
	serviceInstallUpdater = func(string) error {
		t.Fatal("non-Homebrew install enabled updater")
		return nil
	}
	disabled := false
	serviceDisableUpdater = func() error {
		disabled = true
		return nil
	}

	cmd := newServiceCmd()
	cmd.SetArgs([]string{"install"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("service install error = %v", err)
	}
	if !disabled {
		t.Fatal("non-Homebrew install did not remove stale updater")
	}
}

func TestServiceStatusIncludesUpdater(t *testing.T) {
	oldTimerStatus := serviceTimerStatus
	oldMenuBarStatus := serviceMenuBarStatus
	oldUpdaterStatus := serviceUpdaterStatus
	serviceTimerStatus = func() string { return "Installed & Running" }
	serviceMenuBarStatus = func() string { return "Installed & Running" }
	serviceUpdaterStatus = func() string { return "Installed & Running" }
	t.Cleanup(func() {
		serviceTimerStatus = oldTimerStatus
		serviceMenuBarStatus = oldMenuBarStatus
		serviceUpdaterStatus = oldUpdaterStatus
	})

	out := new(bytes.Buffer)
	cmd := newServiceCmd()
	cmd.SetArgs([]string{"status"})
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("service status error = %v", err)
	}
	for _, want := range []string{"Timer: Installed & Running", "Menu Bar: Installed & Running", "Auto Update: Installed & Running"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("service status output = %q, missing %q", out.String(), want)
		}
	}
}
