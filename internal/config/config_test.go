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
	if cfg.WorkStartHour != 9 {
		t.Errorf("WorkStartHour = %d, want 9", cfg.WorkStartHour)
	}
	if cfg.WorkStartMinute != 0 {
		t.Errorf("WorkStartMinute = %d, want 0", cfg.WorkStartMinute)
	}
	if cfg.WorkEndHour != 18 {
		t.Errorf("WorkEndHour = %d, want 18", cfg.WorkEndHour)
	}
	if cfg.WorkEndMinute != 0 {
		t.Errorf("WorkEndMinute = %d, want 0", cfg.WorkEndMinute)
	}
	if cfg.BreakScreenMode != "ask" {
		t.Errorf("BreakScreenMode = %q, want 'ask'", cfg.BreakScreenMode)
	}
	if cfg.TTSEngine != "say" {
		t.Errorf("TTSEngine = %q, want 'say'", cfg.TTSEngine)
	}
	if cfg.TTSModel != "KittenML/kitten-tts-nano-0.8" {
		t.Errorf("TTSModel = %q, want KittenML/kitten-tts-nano-0.8", cfg.TTSModel)
	}
	if cfg.TTSPythonCmd != "python3" {
		t.Errorf("TTSPythonCmd = %q, want 'python3'", cfg.TTSPythonCmd)
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

func TestMergeBreakScreenMode(t *testing.T) {
	yamlData := []byte("break_screen_mode: block\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.BreakScreenMode != "block" {
		t.Errorf("BreakScreenMode = %q, want 'block'", cfg.BreakScreenMode)
	}
}

func TestMergeBreakScreenModeUnset(t *testing.T) {
	yamlData := []byte("work_duration_min: 25\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.BreakScreenMode != "ask" {
		t.Errorf("BreakScreenMode = %q, want 'ask' (default)", cfg.BreakScreenMode)
	}
}

func TestMergeWorkScheduleMinutes(t *testing.T) {
	yamlData := []byte("work_start_hour: 8\nwork_start_minute: 30\nwork_end_hour: 17\nwork_end_minute: 45\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.WorkStartHour != 8 {
		t.Errorf("WorkStartHour = %d, want 8", cfg.WorkStartHour)
	}
	if cfg.WorkStartMinute != 30 {
		t.Errorf("WorkStartMinute = %d, want 30", cfg.WorkStartMinute)
	}
	if cfg.WorkEndHour != 17 {
		t.Errorf("WorkEndHour = %d, want 17", cfg.WorkEndHour)
	}
	if cfg.WorkEndMinute != 45 {
		t.Errorf("WorkEndMinute = %d, want 45", cfg.WorkEndMinute)
	}
}

func TestMergeWorkScheduleMinutesBackwardCompatible(t *testing.T) {
	yamlData := []byte("work_start_hour: 8\nwork_end_hour: 17\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.WorkStartHour != 8 {
		t.Errorf("WorkStartHour = %d, want 8", cfg.WorkStartHour)
	}
	if cfg.WorkStartMinute != 0 {
		t.Errorf("WorkStartMinute = %d, want 0 default when unset", cfg.WorkStartMinute)
	}
	if cfg.WorkEndHour != 17 {
		t.Errorf("WorkEndHour = %d, want 17", cfg.WorkEndHour)
	}
	if cfg.WorkEndMinute != 0 {
		t.Errorf("WorkEndMinute = %d, want 0 default when unset", cfg.WorkEndMinute)
	}
}

func TestMergeWorkScheduleAllowsMidnightHour(t *testing.T) {
	yamlData := []byte("work_start_hour: 0\nwork_start_minute: 30\nwork_end_hour: 0\nwork_end_minute: 45\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.WorkStartHour != 0 {
		t.Errorf("WorkStartHour = %d, want 0", cfg.WorkStartHour)
	}
	if cfg.WorkStartMinute != 30 {
		t.Errorf("WorkStartMinute = %d, want 30", cfg.WorkStartMinute)
	}
	if cfg.WorkEndHour != 0 {
		t.Errorf("WorkEndHour = %d, want 0", cfg.WorkEndHour)
	}
	if cfg.WorkEndMinute != 45 {
		t.Errorf("WorkEndMinute = %d, want 45", cfg.WorkEndMinute)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Override config path for test
	origDir := configDir
	defer func() { configDir = origDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	cfg := Default()
	cfg.BreakScreenMode = "block"
	cfg.WorkDurationMin = 30
	cfg.WorkStartMinute = 15
	cfg.WorkEndMinute = 45

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.BreakScreenMode != "block" {
		t.Errorf("BreakScreenMode = %q, want 'block'", loaded.BreakScreenMode)
	}
	if loaded.WorkDurationMin != 30 {
		t.Errorf("WorkDurationMin = %d, want 30", loaded.WorkDurationMin)
	}
	if loaded.WorkStartMinute != 15 {
		t.Errorf("WorkStartMinute = %d, want 15", loaded.WorkStartMinute)
	}
	if loaded.WorkEndMinute != 45 {
		t.Errorf("WorkEndMinute = %d, want 45", loaded.WorkEndMinute)
	}
}

func TestLoadLegacyScheduleWithoutMinuteFields(t *testing.T) {
	origDir := configDir
	defer func() { configDir = origDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	legacy := []byte("work_start_hour: 8\nwork_end_hour: 17\n")
	if err := os.WriteFile(ConfigPath(), legacy, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.WorkStartHour != 8 {
		t.Errorf("WorkStartHour = %d, want 8", loaded.WorkStartHour)
	}
	if loaded.WorkStartMinute != 0 {
		t.Errorf("WorkStartMinute = %d, want 0", loaded.WorkStartMinute)
	}
	if loaded.WorkEndHour != 17 {
		t.Errorf("WorkEndHour = %d, want 17", loaded.WorkEndHour)
	}
	if loaded.WorkEndMinute != 0 {
		t.Errorf("WorkEndMinute = %d, want 0", loaded.WorkEndMinute)
	}
}

func TestMergeTTSSettings(t *testing.T) {
	yamlData := []byte("tts_engine: kittentts\ntts_model: KittenML/kitten-tts-micro-0.8\ntts_python_cmd: /tmp/venv/bin/python\nvoice: Jasper\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.TTSEngine != "kittentts" {
		t.Errorf("TTSEngine = %q, want kittentts", cfg.TTSEngine)
	}
	if cfg.TTSModel != "KittenML/kitten-tts-micro-0.8" {
		t.Errorf("TTSModel = %q, want KittenML/kitten-tts-micro-0.8", cfg.TTSModel)
	}
	if cfg.TTSPythonCmd != "/tmp/venv/bin/python" {
		t.Errorf("TTSPythonCmd = %q, want /tmp/venv/bin/python", cfg.TTSPythonCmd)
	}
	if cfg.Voice != "Jasper" {
		t.Errorf("Voice = %q, want Jasper", cfg.Voice)
	}
}
