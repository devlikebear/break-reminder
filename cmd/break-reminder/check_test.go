package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestRunCheckOutsideWorkingHoursPreservesPausedLastCheck(t *testing.T) {
	origCfg := cfg
	origNow := nowFunc
	defer func() {
		cfg = origCfg
		nowFunc = origNow
	}()

	cfg = config.Default()
	cfg.WorkDays = nil
	now := time.Unix(1_700_000_000, 0)
	nowFunc = func() time.Time { return now }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	if err := state.Save(statePath, state.State{Mode: "work", Paused: true, PausedAt: now.Add(-time.Minute).Unix(), LastCheck: now.Add(-2 * time.Hour).Unix(), LastUpdateDate: now.Format("2006-01-02")}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := runCheck(); err != nil {
		t.Fatalf("runCheck() error = %v", err)
	}

	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LastCheck != now.Add(-2*time.Hour).Unix() {
		t.Fatalf("LastCheck = %d, want %d", loaded.LastCheck, now.Add(-2*time.Hour).Unix())
	}
	if !loaded.Paused {
		t.Fatal("Paused = false, want true")
	}
}

func TestRunCheckRecoversFromStateLoadFailure(t *testing.T) {
	origCfg := cfg
	origNow := nowFunc
	defer func() {
		cfg = origCfg
		nowFunc = origNow
	}()

	cfg = config.Default()
	cfg.WorkDays = nil
	now := time.Unix(1_700_000_000, 0)
	nowFunc = func() time.Time { return now }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := runCheck(); err == nil {
		t.Fatal("runCheck() error = nil, want state save failure for directory path")
	}

	if err := os.RemoveAll(statePath); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if err := runCheck(); err != nil {
		t.Fatalf("runCheck() after cleanup error = %v", err)
	}

	loaded, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LastCheck != now.Unix() {
		t.Fatalf("LastCheck = %d, want %d", loaded.LastCheck, now.Unix())
	}
}
