package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestRunCheckOutsideWorkingHoursPreservesPausedLastCheck(t *testing.T) {
	origNowFunc := nowFunc
	origCfg := cfg
	defer func() {
		nowFunc = origNowFunc
		cfg = origCfg
	}()

	cfg = config.Default()
	nowFunc = func() time.Time {
		return time.Date(2025, 1, 15, 7, 30, 0, 0, time.Local)
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")

	original := state.State{
		Mode:           "work",
		Paused:         true,
		PausedAt:       1_700_000_060,
		LastCheck:      1_700_000_000,
		LastUpdateDate: "2025-01-15",
	}
	if err := state.Save(statePath, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := runCheck(); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}

	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LastCheck != original.LastCheck {
		t.Fatalf("LastCheck = %d, want %d", loaded.LastCheck, original.LastCheck)
	}
	if !loaded.Paused {
		t.Fatal("Paused = false, want true")
	}
}

func TestRunCheckOutsideWorkingHoursUpdatesUnpausedLastCheck(t *testing.T) {
	origNowFunc := nowFunc
	origCfg := cfg
	defer func() {
		nowFunc = origNowFunc
		cfg = origCfg
	}()

	cfg = config.Default()
	now := time.Date(2025, 1, 15, 7, 30, 0, 0, time.Local)
	nowFunc = func() time.Time { return now }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")

	original := state.State{
		Mode:           "work",
		LastCheck:      1_700_000_000,
		LastUpdateDate: "2025-01-15",
	}
	if err := state.Save(statePath, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := runCheck(); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}

	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LastCheck != now.Unix() {
		t.Fatalf("LastCheck = %d, want %d", loaded.LastCheck, now.Unix())
	}
}
