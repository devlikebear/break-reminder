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

func TestRunCheckRecoversFromStateLoadFailure(t *testing.T) {
	origCfg := cfg
	origNow := nowFunc
	defer func() {
		cfg = origCfg
		nowFunc = origNow
	}()

	cfg = config.Default()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)
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

func TestTimerNeedsTick(t *testing.T) {
	base := config.Default()
	legacy := config.Default()
	legacy.AutoSessionDetect = false

	wednesday := func(hour int) time.Time {
		return time.Date(2025, 1, 15, hour, 0, 0, 0, time.Local)
	}
	sunday := time.Date(2025, 1, 19, 10, 0, 0, 0, time.Local)

	active := state.State{SessionState: state.SessionStateActive}
	stopped := state.State{
		SessionState:   state.SessionStateEnded,
		SessionManual:  true,
		LastUpdateDate: "2025-01-15",
	}
	stoppedYesterday := state.State{
		SessionState:   state.SessionStateEnded,
		SessionManual:  true,
		LastUpdateDate: "2025-01-14",
	}

	tests := []struct {
		name  string
		cfg   config.Config
		state state.State
		now   time.Time
		want  bool
	}{
		{"inside detect window", base, state.State{}, wednesday(8), true},
		{"before detect window", base, state.State{}, wednesday(5), false},
		{"after detect window", base, state.State{}, wednesday(23), false},
		{"weekend without session", base, state.State{}, sunday, false},
		{"running session on a weekend", base, active, sunday, true},
		{"running session after the window closes", base, active, wednesday(23), true},
		{"paused timer at night", base, state.State{Paused: true}, wednesday(3), true},
		{"stopped by hand inside the window", base, stopped, wednesday(10), false},
		{"stopped by hand on a previous day", base, stoppedYesterday, wednesday(10), true},
		{"legacy inside working hours", legacy, state.State{}, wednesday(10), true},
		{"legacy outside working hours", legacy, state.State{}, wednesday(20), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timerNeedsTick(tt.cfg, tt.state, tt.now); got != tt.want {
				t.Errorf("timerNeedsTick() = %v, want %v", got, tt.want)
			}
		})
	}
}
