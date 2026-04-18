package insights

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	pathOverride = filepath.Join(t.TempDir(), "nope.json")

	result, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for missing file, got %v", result)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	pathOverride = filepath.Join(t.TempDir(), "insights.json")

	report := &Report{
		GeneratedAt: "2026-04-17T17:30:00+09:00",
		DailyReport: "오늘 4시간 20분 작업",
		Patterns: []Pattern{
			{Type: "warning", Title: "오후 슬럼프", Description: "D", Suggestion: "S"},
		},
	}

	if err := Save(report); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil report")
	}
	if loaded.DailyReport != "오늘 4시간 20분 작업" {
		t.Errorf("DailyReport mismatch: %q", loaded.DailyReport)
	}
	if len(loaded.Patterns) != 1 {
		t.Errorf("Patterns count = %d, want 1", len(loaded.Patterns))
	}
}

func TestSaveCreatesMissingDirectory(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	pathOverride = filepath.Join(dir, "insights.json")

	report := &Report{GeneratedAt: "2026-04-17T00:00:00Z"}
	if err := Save(report); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(pathOverride); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
