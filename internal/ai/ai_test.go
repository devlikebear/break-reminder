package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("claude")
	if c.CLIName != "claude" {
		t.Errorf("CLIName = %q, want 'claude'", c.CLIName)
	}
	if c.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
}

func TestNewClientCodex(t *testing.T) {
	c := NewClient("codex")
	if c.CLIName != "codex" {
		t.Errorf("CLIName = %q, want 'codex'", c.CLIName)
	}
}

// --- History tests using temp files ---

func TestLoadHistoryMissingFile(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()

	historyPathOverride = filepath.Join(t.TempDir(), "nonexistent.json")

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history != nil {
		t.Errorf("expected nil for missing file, got %v", history)
	}
}

func TestAppendAndLoadHistory(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()

	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	// Append first entry
	s1 := DailySummary{Date: "2026-03-18", WorkMin: 200, BreakMin: 40, Sessions: 4, Activities: 2}
	if err := AppendHistory(s1); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("got %d entries, want 1", len(history))
	}
	if history[0].WorkMin != 200 {
		t.Errorf("WorkMin = %d, want 200", history[0].WorkMin)
	}

	// Append second entry (different date)
	s2 := DailySummary{Date: "2026-03-19", WorkMin: 150, BreakMin: 30, Sessions: 3, Activities: 1}
	if err := AppendHistory(s2); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	history, err = LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("got %d entries, want 2", len(history))
	}
}

func TestAppendHistoryUpdatesSameDate(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()

	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	s1 := DailySummary{Date: "2026-03-18", WorkMin: 100, BreakMin: 20}
	_ = AppendHistory(s1)

	// Update same date
	s2 := DailySummary{Date: "2026-03-18", WorkMin: 200, BreakMin: 40}
	_ = AppendHistory(s2)

	history, _ := LoadHistory()
	if len(history) != 1 {
		t.Fatalf("got %d entries, want 1 (should update, not append)", len(history))
	}
	if history[0].WorkMin != 200 {
		t.Errorf("WorkMin = %d, want 200 (updated)", history[0].WorkMin)
	}
}

func TestLoadHistoryInvalidJSON(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()

	path := filepath.Join(t.TempDir(), "bad.json")
	historyPathOverride = path
	_ = os.WriteFile(path, []byte("not json"), 0o644)

	_, err := LoadHistory()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHistoryFileFormat(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()

	path := filepath.Join(t.TempDir(), "history.json")
	historyPathOverride = path

	_ = AppendHistory(DailySummary{Date: "2026-03-18", WorkMin: 100})

	data, _ := os.ReadFile(path)
	var parsed []DailySummary
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("history file should be valid JSON: %v", err)
	}
}

func TestDailySummaryHourlyWorkPersists(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()
	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	summary := DailySummary{
		Date:       "2026-04-17",
		WorkMin:    280,
		BreakMin:   60,
		Sessions:   7,
		Activities: 3,
		HourlyWork: [24]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 45, 55, 50, 10, 40, 50, 35, 20, 0, 0, 0, 0, 0, 0, 0},
	}
	if err := AppendHistory(summary); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	if history[0].HourlyWork[10] != 55 {
		t.Errorf("HourlyWork[10] = %d, want 55", history[0].HourlyWork[10])
	}
}

func TestDailySummaryBackwardCompatMissingHourly(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()
	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	// Write legacy JSON without hourly_work field
	legacy := `[{"date":"2026-04-16","work_min":200,"break_min":40,"sessions":4,"activities":2}]`
	if err := os.WriteFile(historyPathOverride, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	for i, v := range history[0].HourlyWork {
		if v != 0 {
			t.Errorf("HourlyWork[%d] = %d, want 0 for legacy data", i, v)
		}
	}
}
