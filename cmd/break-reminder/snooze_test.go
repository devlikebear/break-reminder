package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestRunSnoozeEndsBreakAndPostponesNextBreak(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	now := time.Unix(1_700_000_000, 0)

	if err := state.Save(statePath, state.State{
		Mode:                   "break",
		BreakStart:             now.Add(-2 * time.Minute).Unix(),
		LastCheck:              now.Add(-30 * time.Second).Unix(),
		LastUpdateDate:         now.Format("2006-01-02"),
		LastBreakWarningBucket: 2,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stdout := bytes.Buffer{}
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	err = runSnooze(statePath, now, 5*time.Minute)
	_ = w.Close()
	if err != nil {
		t.Fatalf("runSnooze() error = %v", err)
	}
	if _, err := stdout.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	updated, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if updated.Mode != "work" {
		t.Fatalf("Mode = %q, want work", updated.Mode)
	}
	if updated.SnoozeUntil != now.Add(5*time.Minute).Unix() {
		t.Fatalf("SnoozeUntil = %d, want %d", updated.SnoozeUntil, now.Add(5*time.Minute).Unix())
	}
	if updated.BreakStart != 0 {
		t.Fatalf("BreakStart = %d, want 0", updated.BreakStart)
	}
	if !strings.Contains(stdout.String(), "Break snoozed for 5m0s") {
		t.Fatalf("stdout = %q, want snooze confirmation", stdout.String())
	}
}

func TestRunSnoozeRejectsNonBreakStates(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "work", mode: "work", want: "cannot snooze: no active break"},
		{name: "legacy paused mode", mode: "paused", want: "cannot snooze while paused"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "state")
			now := time.Unix(1_700_000_000, 0)
			if err := state.Save(statePath, state.State{Mode: tt.mode, LastUpdateDate: now.Format("2006-01-02")}); err != nil {
				t.Fatalf("Save: %v", err)
			}

			err := runSnooze(statePath, now, 5*time.Minute)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("runSnooze() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRunSnoozeRejectsPausedBreakState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state")
	now := time.Unix(1_700_000_000, 0)
	if err := state.Save(statePath, state.State{Mode: "break", Paused: true, PausedAt: now.Unix(), LastUpdateDate: now.Format("2006-01-02")}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := runSnooze(statePath, now, 5*time.Minute)
	if err == nil || err.Error() != "cannot snooze while paused" {
		t.Fatalf("runSnooze() error = %v, want paused-state rejection", err)
	}
}

func TestRunSnoozeRejectsInvalidDuration(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state")
	now := time.Unix(1_700_000_000, 0)
	if err := state.Save(statePath, state.State{Mode: "break", LastUpdateDate: now.Format("2006-01-02")}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := runSnooze(statePath, now, 0)
	if !errors.Is(err, state.ErrInvalidSnooze) {
		t.Fatalf("runSnooze() error = %v, want %v", err, state.ErrInvalidSnooze)
	}
}

func TestSnoozeCommandAllowsInvalidConfig(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	now := time.Unix(1_700_000_000, 0)
	if err := state.Save(statePath, state.State{Mode: "break", LastUpdateDate: now.Format("2006-01-02")}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	origNow := snoozeNow
	defer func() { snoozeNow = origNow }()
	snoozeNow = func() time.Time { return now }

	cmd := newRootCmd()
	cmd.SetArgs([]string{"snooze"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
}
