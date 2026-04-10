package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-state")

	original := State{
		WorkSeconds:            1800,
		Mode:                   "work",
		LastCheck:              1700000000,
		BreakStart:             0,
		SnoozeUntil:            1700000300,
		Paused:                 true,
		PausedAt:               1700000100,
		TodayWorkSeconds:       3600,
		TodayBreakSeconds:      600,
		LastUpdateDate:         "2025-01-15",
		LastBreakWarningBucket: 2,
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.WorkSeconds != original.WorkSeconds {
		t.Errorf("WorkSeconds = %d, want %d", loaded.WorkSeconds, original.WorkSeconds)
	}
	if loaded.Mode != original.Mode {
		t.Errorf("Mode = %q, want %q", loaded.Mode, original.Mode)
	}
	if loaded.LastCheck != original.LastCheck {
		t.Errorf("LastCheck = %d, want %d", loaded.LastCheck, original.LastCheck)
	}
	if loaded.TodayWorkSeconds != original.TodayWorkSeconds {
		t.Errorf("TodayWorkSeconds = %d, want %d", loaded.TodayWorkSeconds, original.TodayWorkSeconds)
	}
	if loaded.SnoozeUntil != original.SnoozeUntil {
		t.Errorf("SnoozeUntil = %d, want %d", loaded.SnoozeUntil, original.SnoozeUntil)
	}
	if loaded.Paused != original.Paused {
		t.Errorf("Paused = %t, want %t", loaded.Paused, original.Paused)
	}
	if loaded.PausedAt != original.PausedAt {
		t.Errorf("PausedAt = %d, want %d", loaded.PausedAt, original.PausedAt)
	}
	if loaded.TodayBreakSeconds != original.TodayBreakSeconds {
		t.Errorf("TodayBreakSeconds = %d, want %d", loaded.TodayBreakSeconds, original.TodayBreakSeconds)
	}
	if loaded.LastUpdateDate != original.LastUpdateDate {
		t.Errorf("LastUpdateDate = %q, want %q", loaded.LastUpdateDate, original.LastUpdateDate)
	}
	if loaded.LastBreakWarningBucket != original.LastBreakWarningBucket {
		t.Errorf("LastBreakWarningBucket = %d, want %d", loaded.LastBreakWarningBucket, original.LastBreakWarningBucket)
	}
}

func TestLoadBashFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bash-state")

	content := `WORK_SECONDS=2400
MODE=break
LAST_CHECK=1700000100
BREAK_START=1700000000
SNOOZE_UNTIL=1700000300
PAUSED=true
PAUSED_AT=1700000200
TODAY_WORK_SECONDS=7200
TODAY_BREAK_SECONDS=1200
LAST_UPDATE_DATE=2025-01-15
LAST_BREAK_WARNING_BUCKET=3
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.WorkSeconds != 2400 {
		t.Errorf("WorkSeconds = %d, want 2400", s.WorkSeconds)
	}
	if s.Mode != "break" {
		t.Errorf("Mode = %q, want %q", s.Mode, "break")
	}
	if s.BreakStart != 1700000000 {
		t.Errorf("BreakStart = %d, want 1700000000", s.BreakStart)
	}
	if s.SnoozeUntil != 1700000300 {
		t.Errorf("SnoozeUntil = %d, want 1700000300", s.SnoozeUntil)
	}
	if !s.Paused {
		t.Fatal("Paused = false, want true")
	}
	if s.PausedAt != 1700000200 {
		t.Errorf("PausedAt = %d, want 1700000200", s.PausedAt)
	}
	if s.LastBreakWarningBucket != 3 {
		t.Errorf("LastBreakWarningBucket = %d, want 3", s.LastBreakWarningBucket)
	}
}

func TestLoadMissing(t *testing.T) {
	s, err := Load("/nonexistent/path")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if s.Mode != "work" {
		t.Errorf("Mode = %q, want default %q", s.Mode, "work")
	}
}

func TestEnterBreakResetsWarningBucket(t *testing.T) {
	updated := (State{
		Mode:                   "work",
		WorkSeconds:            1800,
		BreakStart:             123,
		SnoozeUntil:            789,
		LastBreakWarningBucket: 3,
	}).EnterBreak(456)

	if updated.Mode != "break" {
		t.Fatalf("Mode = %q, want break", updated.Mode)
	}
	if updated.BreakStart != 456 {
		t.Fatalf("BreakStart = %d, want 456", updated.BreakStart)
	}
	if updated.WorkSeconds != 0 {
		t.Fatalf("WorkSeconds = %d, want 0", updated.WorkSeconds)
	}
	if updated.SnoozeUntil != 0 {
		t.Fatalf("SnoozeUntil = %d, want 0", updated.SnoozeUntil)
	}
	if updated.LastBreakWarningBucket != 0 {
		t.Fatalf("LastBreakWarningBucket = %d, want 0", updated.LastBreakWarningBucket)
	}
}

func TestResumeWithMissingPausedAtResetsAnchorsSafely(t *testing.T) {
	updated := (State{
		Mode:       "break",
		Paused:     true,
		PausedAt:   0,
		LastCheck:  100,
		BreakStart: 80,
	}).Resume(250)

	if updated.Paused {
		t.Fatal("Paused = true, want false")
	}
	if updated.PausedAt != 0 {
		t.Fatalf("PausedAt = %d, want 0", updated.PausedAt)
	}
	if updated.LastCheck != 250 {
		t.Fatalf("LastCheck = %d, want 250", updated.LastCheck)
	}
	if updated.BreakStart != 250 {
		t.Fatalf("BreakStart = %d, want 250", updated.BreakStart)
	}
}

func TestPauseAndResumeRoundTrip(t *testing.T) {
	original := State{
		Mode:           "break",
		LastCheck:      1_700_000_000,
		BreakStart:     1_699_999_400,
		LastUpdateDate: "2025-01-15",
	}

	paused := original.Pause(1_700_000_060)
	if !paused.Paused {
		t.Fatal("Pause() should mark the state as paused")
	}
	if paused.PausedAt != 1_700_000_060 {
		t.Fatalf("PausedAt = %d, want %d", paused.PausedAt, 1_700_000_060)
	}

	resumed := paused.Resume(1_700_000_660)
	if resumed.Paused {
		t.Fatal("Resume() should clear paused state")
	}
	if resumed.PausedAt != 0 {
		t.Fatalf("PausedAt = %d, want 0", resumed.PausedAt)
	}
	if resumed.LastCheck != 1_700_000_660 {
		t.Fatalf("LastCheck = %d, want %d", resumed.LastCheck, 1_700_000_660)
	}
	if resumed.BreakStart != 1_700_000_000 {
		t.Fatalf("BreakStart = %d, want %d", resumed.BreakStart, 1_700_000_000)
	}
}

func TestPauseSettlesElapsedWorkBeforeFreezing(t *testing.T) {
	original := State{
		Mode:             "work",
		WorkSeconds:      900,
		TodayWorkSeconds: 3600,
		LastCheck:        1_700_000_000,
		LastUpdateDate:   "2025-01-15",
	}

	paused := original.Pause(1_700_000_120)

	if paused.WorkSeconds != 1020 {
		t.Fatalf("WorkSeconds = %d, want 1020", paused.WorkSeconds)
	}
	if paused.TodayWorkSeconds != 3720 {
		t.Fatalf("TodayWorkSeconds = %d, want 3720", paused.TodayWorkSeconds)
	}
	if paused.LastCheck != 1_700_000_120 {
		t.Fatalf("LastCheck = %d, want 1700000120", paused.LastCheck)
	}
	if !paused.Paused {
		t.Fatal("Pause() should mark the state as paused")
	}
}

func TestPauseSettlesElapsedBreakAcrossMidnight(t *testing.T) {
	original := State{
		Mode:              "break",
		BreakStart:        time.Date(2025, 1, 15, 23, 45, 0, 0, time.Local).Unix(),
		LastCheck:         time.Date(2025, 1, 15, 23, 59, 0, 0, time.Local).Unix(),
		TodayWorkSeconds:  5000,
		TodayBreakSeconds: 900,
		LastUpdateDate:    "2025-01-15",
	}

	pausedAt := time.Date(2025, 1, 16, 0, 1, 0, 0, time.Local).Unix()
	paused := original.Pause(pausedAt)

	if paused.TodayBreakSeconds != 60 {
		t.Fatalf("TodayBreakSeconds = %d, want 60", paused.TodayBreakSeconds)
	}
	if paused.LastUpdateDate != "2025-01-16" {
		t.Fatalf("LastUpdateDate = %q, want 2025-01-16", paused.LastUpdateDate)
	}
	if paused.LastCheck != pausedAt {
		t.Fatalf("LastCheck = %d, want %d", paused.LastCheck, pausedAt)
	}
	if !paused.Paused {
		t.Fatal("Pause() should mark the state as paused")
	}
}

func TestPauseAlreadyPausedIsNoOp(t *testing.T) {
	paused := State{Mode: "work", Paused: true, PausedAt: 123}.Pause(456)
	if paused.PausedAt != 123 {
		t.Fatalf("PausedAt = %d, want 123", paused.PausedAt)
	}

	resumed := (State{Mode: "work", LastCheck: 789}).Resume(999)
	if resumed.LastCheck != 789 {
		t.Fatalf("LastCheck = %d, want 789", resumed.LastCheck)
	}
	if resumed.Paused {
		t.Fatal("Resume() should not mark an unpaused state as paused")
	}
}

func TestSnoozeBreakEndsBreakAndDefersNextWorkCycle(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	updated, err := (State{
		Mode:                   "break",
		BreakStart:             now.Add(-2 * time.Minute).Unix(),
		LastCheck:              now.Add(-30 * time.Second).Unix(),
		WorkSeconds:            900,
		SnoozeUntil:            now.Add(-time.Minute).Unix(),
		TodayBreakSeconds:      120,
		LastUpdateDate:         now.Format("2006-01-02"),
		LastBreakWarningBucket: 2,
	}).SnoozeBreak(now, 5*time.Minute)
	if err != nil {
		t.Fatalf("SnoozeBreak() error = %v", err)
	}
	if updated.Mode != "work" {
		t.Fatalf("Mode = %q, want work", updated.Mode)
	}
	if updated.BreakStart != 0 {
		t.Fatalf("BreakStart = %d, want 0", updated.BreakStart)
	}
	if updated.LastCheck != now.Unix() {
		t.Fatalf("LastCheck = %d, want %d", updated.LastCheck, now.Unix())
	}
	if updated.SnoozeUntil != now.Add(5*time.Minute).Unix() {
		t.Fatalf("SnoozeUntil = %d, want %d", updated.SnoozeUntil, now.Add(5*time.Minute).Unix())
	}
	if updated.WorkSeconds != 0 {
		t.Fatalf("WorkSeconds = %d, want 0", updated.WorkSeconds)
	}
	if updated.LastBreakWarningBucket != 0 {
		t.Fatalf("LastBreakWarningBucket = %d, want 0", updated.LastBreakWarningBucket)
	}
	if updated.TodayBreakSeconds != 150 {
		t.Fatalf("TodayBreakSeconds = %d, want 150", updated.TodayBreakSeconds)
	}
}

func TestSnoozeBreakRollsOverDailyBreakTotalsBeforeAccumulating(t *testing.T) {
	now := time.Date(2025, 1, 16, 0, 1, 0, 0, time.Local)
	updated, err := (State{
		Mode:              "break",
		BreakStart:        time.Date(2025, 1, 15, 23, 58, 0, 0, time.Local).Unix(),
		LastCheck:         now.Add(-60 * time.Second).Unix(),
		TodayWorkSeconds:  5000,
		TodayBreakSeconds: 900,
		LastUpdateDate:    "2025-01-15",
	}).SnoozeBreak(now, 5*time.Minute)
	if err != nil {
		t.Fatalf("SnoozeBreak() error = %v", err)
	}
	if updated.LastUpdateDate != "2025-01-16" {
		t.Fatalf("LastUpdateDate = %q, want 2025-01-16", updated.LastUpdateDate)
	}
	if updated.TodayWorkSeconds != 0 {
		t.Fatalf("TodayWorkSeconds = %d, want 0", updated.TodayWorkSeconds)
	}
	if updated.TodayBreakSeconds != 60 {
		t.Fatalf("TodayBreakSeconds = %d, want 60", updated.TodayBreakSeconds)
	}
}

func TestSnoozeBreakRejectsInvalidModes(t *testing.T) {
	tests := []struct {
		name  string
		state State
		err   error
	}{
		{name: "work", state: State{Mode: "work"}, err: ErrBreakNotActive},
		{name: "legacy paused mode", state: State{Mode: "paused"}, err: ErrStatePaused},
		{name: "paused break flag", state: State{Mode: "break", Paused: true, PausedAt: 123}, err: ErrStatePaused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.state.SnoozeBreak(time.Unix(1_700_000_000, 0), 5*time.Minute)
			if !errors.Is(err, tt.err) {
				t.Fatalf("SnoozeBreak() error = %v, want %v", err, tt.err)
			}
		})
	}
}
