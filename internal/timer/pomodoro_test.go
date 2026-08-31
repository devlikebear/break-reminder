package timer

import (
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func pomodoroConfig() config.Config {
	cfg := config.Default()
	cfg.TimerMode = config.TimerModePomodoro
	return cfg // 25 min work, 5 min break, 15 min long break every 4
}

func TestPomodoroBreakTriggersAtPomodoroWorkDuration(t *testing.T) {
	cfg := pomodoroConfig()
	now := at(10, 0)
	s := activeState(now)
	s.WorkSeconds = 25*60 - 30

	result := Tick(cfg, s, now, 5)

	if result.State.Mode != "break" {
		t.Fatalf("Mode = %q, want break after 25 minutes of work", result.State.Mode)
	}
	if result.State.PomodoroCount != 1 || result.State.TodayPomodoros != 1 {
		t.Errorf("PomodoroCount/TodayPomodoros = %d/%d, want 1/1",
			result.State.PomodoroCount, result.State.TodayPomodoros)
	}
}

func TestClassicModeDoesNotCountPomodoros(t *testing.T) {
	cfg := config.Default()
	now := at(10, 0)
	s := activeState(now)
	s.WorkSeconds = 50*60 - 30

	result := Tick(cfg, s, now, 5)

	if result.State.Mode != "break" {
		t.Fatalf("Mode = %q, want break", result.State.Mode)
	}
	if result.State.PomodoroCount != 0 || result.State.TodayPomodoros != 0 {
		t.Errorf("classic mode counted pomodoros: %d/%d",
			result.State.PomodoroCount, result.State.TodayPomodoros)
	}
}

func TestPomodoroShortBreakEndsAfterShortDuration(t *testing.T) {
	cfg := pomodoroConfig()
	now := at(10, 30)
	s := activeState(now)
	s.Mode = "break"
	s.PomodoroCount = 1
	s.BreakStart = now.Add(-5 * time.Minute).Unix()

	result := Tick(cfg, s, now, 5)

	if result.State.Mode != "work" {
		t.Fatalf("Mode = %q, want work after a 5 minute short break", result.State.Mode)
	}
	if result.State.PomodoroCount != 1 {
		t.Errorf("PomodoroCount = %d, want the cycle to continue at 1", result.State.PomodoroCount)
	}
}

func TestPomodoroFourthBreakIsLongAndClosesTheCycle(t *testing.T) {
	cfg := pomodoroConfig()
	now := at(12, 0)

	// Finishing the fourth focus block starts the long break.
	work := activeState(now)
	work.PomodoroCount = 3
	work.WorkSeconds = 25*60 - 30
	entered := Tick(cfg, work, now, 5)
	if entered.State.Mode != "break" || entered.State.PomodoroCount != 4 {
		t.Fatalf("Mode/PomodoroCount = %q/%d, want break/4", entered.State.Mode, entered.State.PomodoroCount)
	}

	// A short break's worth of time is not enough for the long break.
	sixMin := now.Add(6 * time.Minute)
	s := entered.State
	s.LastCheck = sixMin.Add(-60 * time.Second).Unix()
	stillResting := Tick(cfg, s, sixMin, 5)
	if stillResting.State.Mode != "break" {
		t.Fatalf("Mode = %q, want the long break to continue after 6 minutes", stillResting.State.Mode)
	}

	// After 15 minutes the long break ends and the cycle restarts.
	fifteenMin := now.Add(15 * time.Minute)
	s = stillResting.State
	s.LastCheck = fifteenMin.Add(-60 * time.Second).Unix()
	done := Tick(cfg, s, fifteenMin, 5)
	if done.State.Mode != "work" {
		t.Fatalf("Mode = %q, want work after the long break", done.State.Mode)
	}
	if done.State.PomodoroCount != 0 {
		t.Errorf("PomodoroCount = %d, want 0 after the long break", done.State.PomodoroCount)
	}
	if done.State.TodayPomodoros != 1 {
		t.Errorf("TodayPomodoros = %d, want the daily total to survive the cycle reset", done.State.TodayPomodoros)
	}
}

func TestPomodoroCountResetsOnNewDay(t *testing.T) {
	cfg := pomodoroConfig()
	now := at(9, 0)
	s := activeState(now)
	s.LastUpdateDate = "2025-01-14"
	s.TodayPomodoros = 8
	s.TodayWorkSeconds = 7200

	result := Tick(cfg, s, now, 5)

	if result.State.TodayPomodoros != 0 {
		t.Errorf("TodayPomodoros = %d, want 0 on a new day", result.State.TodayPomodoros)
	}
}

func TestPomodoroSessionStartResetsCycle(t *testing.T) {
	now := at(9, 0)
	s := state.State{PomodoroCount: 3, TodayPomodoros: 3}

	started := s.StartSession(now.Unix(), true)

	if started.PomodoroCount != 0 {
		t.Errorf("PomodoroCount = %d, want a fresh cycle", started.PomodoroCount)
	}
	if started.TodayPomodoros != 3 {
		t.Errorf("TodayPomodoros = %d, want the daily total kept", started.TodayPomodoros)
	}
}
