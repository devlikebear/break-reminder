package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-state")

	original := State{
		WorkSeconds:            1800,
		Mode:                   "work",
		LastCheck:              1700000000,
		BreakStart:             0,
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

	// Simulate bash script's state file format
	content := `WORK_SECONDS=2400
MODE=break
LAST_CHECK=1700000100
BREAK_START=1700000000
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
	if resumed.LastCheck != 1_700_000_600 {
		t.Fatalf("LastCheck = %d, want %d", resumed.LastCheck, 1_700_000_600)
	}
	if resumed.BreakStart != 1_700_000_000 {
		t.Fatalf("BreakStart = %d, want %d", resumed.BreakStart, 1_700_000_000)
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
