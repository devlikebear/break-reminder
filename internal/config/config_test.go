package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.WorkDurationMin != 50 {
		t.Errorf("WorkDurationMin = %d, want 50", cfg.WorkDurationMin)
	}
	if cfg.BreakDurationMin != 10 {
		t.Errorf("BreakDurationMin = %d, want 10", cfg.BreakDurationMin)
	}
	if cfg.WorkDurationSec() != 3000 {
		t.Errorf("WorkDurationSec = %d, want 3000", cfg.WorkDurationSec())
	}
	if len(cfg.WorkDays) != 5 {
		t.Errorf("WorkDays = %v, want 5 days", cfg.WorkDays)
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".config", "break-reminder")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	custom := Config{
		WorkDurationMin:  25,
		BreakDurationMin: 5,
		Voice:            "Samantha",
	}
	data, err := yaml.Marshal(&custom)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Note: Load() uses hardcoded path, so we test merge directly
	// Build raw map to simulate what YAML unmarshal produces
	var raw map[string]any
	_ = yaml.Unmarshal(data, &raw)

	cfg := Default()
	merge(&cfg, &custom, raw)

	if cfg.WorkDurationMin != 25 {
		t.Errorf("WorkDurationMin = %d, want 25", cfg.WorkDurationMin)
	}
	if cfg.BreakDurationMin != 5 {
		t.Errorf("BreakDurationMin = %d, want 5", cfg.BreakDurationMin)
	}
	if cfg.Voice != "Samantha" {
		t.Errorf("Voice = %q, want Samantha", cfg.Voice)
	}
	// Unchanged defaults
	if cfg.IdleThresholdSec != 120 {
		t.Errorf("IdleThresholdSec = %d, want 120 (unchanged default)", cfg.IdleThresholdSec)
	}
}

func TestMergeBooleans(t *testing.T) {
	yamlData := []byte("ai_enabled: true\ntts_enabled: false\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if !cfg.AIEnabled {
		t.Error("AIEnabled should be true after merge")
	}
	if cfg.TTSEnabled {
		t.Error("TTSEnabled should be false after merge (explicitly set)")
	}
	// NotificationsEnabled not in YAML, should keep default (true)
	if !cfg.NotificationsEnabled {
		t.Error("NotificationsEnabled should remain true (not in YAML)")
	}
}
