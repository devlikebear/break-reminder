package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func runSessionCmd(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newRootCmd()
	cmd.SetArgs(args)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v Execute() error = %v", args, err)
	}
	return out.String()
}

func setupSessionTest(t *testing.T, cfg config.Config, now time.Time) string {
	t.Helper()

	origLoadConfig := loadConfig
	origNowFunc := nowFunc
	t.Cleanup(func() {
		loadConfig = origLoadConfig
		nowFunc = origNowFunc
	})

	loadConfig = func() (config.Config, error) { return cfg, nil }
	nowFunc = func() time.Time { return now }

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	return filepath.Join(tmpHome, ".break-reminder-state")
}

func TestStartCommandOpensManualSession(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	statePath := setupSessionTest(t, config.Default(), now)

	out := runSessionCmd(t, "start")
	if !strings.Contains(out, "Work session started") {
		t.Fatalf("start output = %q, want a start confirmation", out)
	}

	s, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !s.IsSessionActive() {
		t.Fatalf("SessionState = %q, want active", s.SessionState)
	}
	if !s.SessionManual {
		t.Error("start should mark the session as manual")
	}
	if s.SessionStart != now.Unix() {
		t.Errorf("SessionStart = %d, want %d", s.SessionStart, now.Unix())
	}
}

func TestStartCommandOnRunningSessionKeepsItUnchanged(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	statePath := setupSessionTest(t, config.Default(), now)

	runSessionCmd(t, "start")
	nowFunc = func() time.Time { return now.Add(10 * time.Minute) }
	out := runSessionCmd(t, "start")

	if !strings.Contains(out, "already running") {
		t.Fatalf("second start output = %q, want an already-running notice", out)
	}
	s, _ := state.Load(statePath)
	if s.SessionStart != now.Unix() {
		t.Errorf("SessionStart = %d, want the original %d", s.SessionStart, now.Unix())
	}
}

func TestStopCommandEndsSessionAndHoldsReminders(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	statePath := setupSessionTest(t, config.Default(), now)

	runSessionCmd(t, "start")

	if err := state.Update(statePath, func(s state.State) (state.State, error) {
		s.TodayWorkSeconds = 3600
		s.TodayBreakSeconds = 600
		return s, nil
	}); err != nil {
		t.Fatalf("seed daily totals: %v", err)
	}

	nowFunc = func() time.Time { return now.Add(90 * time.Minute) }
	out := runSessionCmd(t, "stop")

	if !strings.Contains(out, "Work session stopped") {
		t.Fatalf("stop output = %q, want a stop confirmation", out)
	}
	if !strings.Contains(out, "Work 1h") {
		t.Errorf("stop output = %q, want today's totals", out)
	}

	s, _ := state.Load(statePath)
	if s.SessionState != state.SessionStateEnded {
		t.Fatalf("SessionState = %q, want ended", s.SessionState)
	}
	if !s.SessionManual {
		t.Error("stop should hold automatic detection with the manual flag")
	}
	if s.TodayWorkSeconds != 3600 {
		t.Errorf("TodayWorkSeconds = %d, want the daily total preserved", s.TodayWorkSeconds)
	}
}

func TestStopCommandWithoutSessionStillHoldsReminders(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	statePath := setupSessionTest(t, config.Default(), now)

	out := runSessionCmd(t, "stop")

	if !strings.Contains(out, "No work session was running") {
		t.Fatalf("stop output = %q, want the no-session notice", out)
	}
	s, _ := state.Load(statePath)
	if s.SessionState != state.SessionStateEnded || !s.SessionManual {
		t.Fatalf("state = %q (manual=%t), want a manual stop", s.SessionState, s.SessionManual)
	}
}

func TestStartAfterStopReopensTheSession(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	statePath := setupSessionTest(t, config.Default(), now)

	runSessionCmd(t, "stop")
	nowFunc = func() time.Time { return now.Add(30 * time.Minute) }
	runSessionCmd(t, "start")

	s, _ := state.Load(statePath)
	if !s.IsSessionActive() {
		t.Fatalf("SessionState = %q, want active after an explicit start", s.SessionState)
	}
	if s.SessionStart != now.Add(30*time.Minute).Unix() {
		t.Errorf("SessionStart = %d, want %d", s.SessionStart, now.Add(30*time.Minute).Unix())
	}
}

func TestPomodoroOnAndOffUpdateConfig(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setupSessionTest(t, config.Default(), now)

	out := runSessionCmd(t, "pomodoro", "on", "--work", "30", "--every", "3")
	if !strings.Contains(out, "Pomodoro mode on") {
		t.Fatalf("pomodoro on output = %q", out)
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if !saved.PomodoroEnabled() {
		t.Fatalf("TimerMode = %q, want pomodoro", saved.TimerMode)
	}
	if saved.PomodoroWorkMin != 30 || saved.PomodoroLongBreakEvery != 3 {
		t.Errorf("pomodoro settings = %d/%d, want 30/3", saved.PomodoroWorkMin, saved.PomodoroLongBreakEvery)
	}

	// The saved config becomes the base for the next command run.
	loadConfig = func() (config.Config, error) { return config.Load() }
	out = runSessionCmd(t, "pomodoro", "off")
	if !strings.Contains(out, "Classic mode on") {
		t.Fatalf("pomodoro off output = %q", out)
	}
	saved, err = config.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if saved.PomodoroEnabled() {
		t.Errorf("TimerMode = %q, want classic", saved.TimerMode)
	}
	if saved.PomodoroWorkMin != 30 {
		t.Errorf("PomodoroWorkMin = %d, want the custom value kept", saved.PomodoroWorkMin)
	}
}
