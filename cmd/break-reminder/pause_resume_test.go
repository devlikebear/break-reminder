package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestPauseAndResumeCommands(t *testing.T) {
	origLoadConfig := loadConfig
	origNowFunc := nowFunc
	defer func() {
		loadConfig = origLoadConfig
		nowFunc = origNowFunc
	}()

	loadConfig = func() (config.Config, error) { return config.Default(), nil }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")

	initial := state.State{
		Mode:           "break",
		LastCheck:      1_700_000_000,
		BreakStart:     1_699_999_400,
		LastUpdateDate: "2025-01-15",
	}
	if err := state.Save(statePath, initial); err != nil {
		t.Fatalf("Save: %v", err)
	}

	nowFunc = func() time.Time { return time.Unix(1_700_000_060, 0) }
	pauseCmd := newRootCmd()
	pauseCmd.SetArgs([]string{"pause"})
	pauseOut := new(bytes.Buffer)
	pauseCmd.SetOut(pauseOut)
	pauseCmd.SetErr(new(bytes.Buffer))
	if err := pauseCmd.Execute(); err != nil {
		t.Fatalf("pause Execute() error = %v", err)
	}
	if !strings.Contains(pauseOut.String(), "Timer paused") {
		t.Fatalf("pause output = %q, want pause confirmation", pauseOut.String())
	}

	paused, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load paused state: %v", err)
	}
	if !paused.Paused {
		t.Fatal("pause command should persist paused=true")
	}
	if paused.PausedAt != 1_700_000_060 {
		t.Fatalf("PausedAt = %d, want %d", paused.PausedAt, 1_700_000_060)
	}

	nowFunc = func() time.Time { return time.Unix(1_700_000_660, 0) }
	resumeCmd := newRootCmd()
	resumeCmd.SetArgs([]string{"resume"})
	resumeOut := new(bytes.Buffer)
	resumeCmd.SetOut(resumeOut)
	resumeCmd.SetErr(new(bytes.Buffer))
	if err := resumeCmd.Execute(); err != nil {
		t.Fatalf("resume Execute() error = %v", err)
	}
	if !strings.Contains(resumeOut.String(), "Timer resumed") {
		t.Fatalf("resume output = %q, want resume confirmation", resumeOut.String())
	}

	resumed, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load resumed state: %v", err)
	}
	if resumed.Paused {
		t.Fatal("resume command should clear paused=true")
	}
	if resumed.LastCheck != 1_700_000_660 {
		t.Fatalf("LastCheck = %d, want %d", resumed.LastCheck, 1_700_000_660)
	}
	if resumed.BreakStart != 1_700_000_000 {
		t.Fatalf("BreakStart = %d, want %d", resumed.BreakStart, 1_700_000_000)
	}
}

func TestPauseAlreadyPausedIsSafe(t *testing.T) {
	origLoadConfig := loadConfig
	origNowFunc := nowFunc
	defer func() {
		loadConfig = origLoadConfig
		nowFunc = origNowFunc
	}()

	loadConfig = func() (config.Config, error) { return config.Default(), nil }
	nowFunc = func() time.Time { return time.Unix(1_700_000_999, 0) }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	if err := os.WriteFile(statePath, []byte("MODE=work\nLAST_CHECK=1700000000\nPAUSED=true\nPAUSED_AT=1700000060\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"pause"})
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pause Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "already paused") {
		t.Fatalf("pause output = %q, want already-paused confirmation", out.String())
	}

	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.PausedAt != 1_700_000_060 {
		t.Fatalf("PausedAt = %d, want %d", loaded.PausedAt, 1_700_000_060)
	}
}
