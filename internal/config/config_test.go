package config

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadRejectsInvalidWorkSchedule(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "negative start minute",
			yaml:    "work_start_minute: -1\n",
			wantErr: "work_start_minute must be between 0 and 59",
		},
		{
			name:    "end minute above 59",
			yaml:    "work_end_minute: 75\n",
			wantErr: "work_end_minute must be between 0 and 59",
		},
		{
			name:    "start hour above 23",
			yaml:    "work_start_hour: 25\n",
			wantErr: "work_start_hour must be between 0 and 23",
		},
		{
			name:    "end hour negative",
			yaml:    "work_end_hour: -1\n",
			wantErr: "work_end_hour must be between 0 and 23",
		},
		{
			name:    "workday window must increase",
			yaml:    "work_start_hour: 17\nwork_end_hour: 17\n",
			wantErr: "work schedule must end after it starts",
		},
		{
			name:    "workday window cannot wrap past midnight",
			yaml:    "work_start_hour: 22\nwork_end_hour: 6\n",
			wantErr: "work schedule must end after it starts",
		},
	}

	origDir := configDir
	defer func() { configDir = origDir }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir = tmpDir
			if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}

			if err := os.WriteFile(ConfigPath(), []byte(tt.yaml), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Load() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadReturnsMergedConfigWhenScheduleInvalid(t *testing.T) {
	origDir := configDir
	defer func() { configDir = origDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	invalidButUseful := []byte("voice: Samantha\nnotifications_enabled: false\nwork_end_minute: 75\n")
	if err := os.WriteFile(ConfigPath(), invalidButUseful, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want schedule validation error")
	}
	if err.Error() != "work_end_minute must be between 0 and 59" {
		t.Fatalf("Load() error = %q, want schedule validation error", err.Error())
	}
	if loaded.Voice != "Samantha" {
		t.Fatalf("Voice = %q, want Samantha", loaded.Voice)
	}
	if loaded.NotificationsEnabled {
		t.Fatal("NotificationsEnabled = true, want false from existing config")
	}
}

func TestApplyYAMLChangesRejectsInvalidScheduleAndPreservesInput(t *testing.T) {
	base := Default()
	base.Voice = "Samantha"
	base.NotificationsEnabled = false

	updated, err := ApplyYAMLChanges(base, []byte("work_end_minute: 75\n"))
	if err == nil {
		t.Fatal("ApplyYAMLChanges() error = nil, want schedule validation error")
	}
	if err.Error() != "work_end_minute must be between 0 and 59" {
		t.Fatalf("ApplyYAMLChanges() error = %q, want schedule validation error", err.Error())
	}
	if !reflect.DeepEqual(updated, base) {
		t.Fatalf("ApplyYAMLChanges() returned mutated config: got %+v want %+v", updated, base)
	}
}

func TestApplyYAMLChangesMergesValidChanges(t *testing.T) {
	base := Default()
	base.Voice = "Samantha"
	base.NotificationsEnabled = false

	updated, err := ApplyYAMLChanges(base, []byte("work_start_minute: 30\nai_enabled: true\n"))
	if err != nil {
		t.Fatalf("ApplyYAMLChanges() error = %v", err)
	}
	if updated.WorkStartMinute != 30 {
		t.Fatalf("WorkStartMinute = %d, want 30", updated.WorkStartMinute)
	}
	if !updated.AIEnabled {
		t.Fatal("AIEnabled = false, want true")
	}
	if updated.Voice != "Samantha" {
		t.Fatalf("Voice = %q, want Samantha", updated.Voice)
	}
	if updated.NotificationsEnabled {
		t.Fatal("NotificationsEnabled = true, want preserved false")
	}
}

func TestApplyYAMLChangesRejectsUnknownKeys(t *testing.T) {
	base := Default()

	updated, err := ApplyYAMLChanges(base, []byte("changes:\n  work_start_minute: 30\n"))
	if err == nil {
		t.Fatal("ApplyYAMLChanges() error = nil, want unknown key error")
	}
	if err.Error() != "unknown config key \"changes\"" {
		t.Fatalf("ApplyYAMLChanges() error = %q, want unknown key error", err.Error())
	}
	if !reflect.DeepEqual(updated, base) {
		t.Fatalf("ApplyYAMLChanges() returned mutated config: got %+v want %+v", updated, base)
	}
}

func TestApplyYAMLChangesRejectsNullScheduleFields(t *testing.T) {
	base := Default()
	base.WorkStartHour = 9
	base.WorkStartMinute = 15
	base.WorkEndHour = 18
	base.WorkEndMinute = 45

	updated, err := ApplyYAMLChanges(base, []byte("work_start_hour: null\nwork_start_minute: null\nwork_end_hour: null\nwork_end_minute: null\n"))
	if err != nil {
		t.Fatalf("ApplyYAMLChanges() error = %v", err)
	}
	if !reflect.DeepEqual(updated, base) {
		t.Fatalf("ApplyYAMLChanges() returned mutated config for null schedule fields: got %+v want %+v", updated, base)
	}
}

func TestApplyYAMLChangesRejectsEmptyChanges(t *testing.T) {
	base := Default()

	updated, err := ApplyYAMLChanges(base, []byte("# nothing to apply\n"))
	if err == nil {
		t.Fatal("ApplyYAMLChanges() error = nil, want empty change error")
	}
	if err.Error() != "no config changes found" {
		t.Fatalf("ApplyYAMLChanges() error = %q, want no config changes found", err.Error())
	}
	if !reflect.DeepEqual(updated, base) {
		t.Fatalf("ApplyYAMLChanges() returned mutated config: got %+v want %+v", updated, base)
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

func TestMergeTTSAPIKey(t *testing.T) {
	yamlData := []byte("tts_engine: gemini\ntts_api_key: sk-secret\nvoice: Zephyr\n")

	var fileCfg Config
	_ = yaml.Unmarshal(yamlData, &fileCfg)

	var raw map[string]any
	_ = yaml.Unmarshal(yamlData, &raw)

	cfg := Default()
	merge(&cfg, &fileCfg, raw)

	if cfg.TTSEngine != "gemini" {
		t.Errorf("TTSEngine = %q, want gemini", cfg.TTSEngine)
	}
	if cfg.TTSAPIKey != "sk-secret" {
		t.Errorf("TTSAPIKey = %q, want sk-secret", cfg.TTSAPIKey)
	}
	if cfg.Voice != "Zephyr" {
		t.Errorf("Voice = %q, want Zephyr", cfg.Voice)
	}
}

func TestApplyYAMLChangesAcceptsTTSAPIKey(t *testing.T) {
	base := Default()

	updated, err := ApplyYAMLChanges(base, []byte("tts_api_key: new-key\n"))
	if err != nil {
		t.Fatalf("ApplyYAMLChanges() error = %v", err)
	}
	if updated.TTSAPIKey != "new-key" {
		t.Fatalf("TTSAPIKey = %q, want new-key", updated.TTSAPIKey)
	}
}

func TestThemeDefaultAuto(t *testing.T) {
	cfg := Default()
	if cfg.Theme != "auto" {
		t.Errorf("Theme default = %q, want 'auto'", cfg.Theme)
	}
}

func TestThemeMergePreservesDefault(t *testing.T) {
	base := Default()
	src := Config{} // empty — should not override
	raw := map[string]any{}
	merge(&base, &src, raw)
	if base.Theme != "auto" {
		t.Errorf("Theme = %q after empty merge, want 'auto'", base.Theme)
	}
}

func TestThemeMergeOverride(t *testing.T) {
	base := Default()
	src := Config{Theme: "dark"}
	raw := map[string]any{"theme": "dark"}
	merge(&base, &src, raw)
	if base.Theme != "dark" {
		t.Errorf("Theme = %q, want 'dark'", base.Theme)
	}
}

func mergeYAML(t *testing.T, yamlData string) Config {
	t.Helper()

	var fileCfg Config
	if err := yaml.Unmarshal([]byte(yamlData), &fileCfg); err != nil {
		t.Fatalf("Unmarshal typed: %v", err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(yamlData), &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}

	cfg := Default()
	merge(&cfg, &fileCfg, raw)
	return cfg
}

func TestSessionDetectionDefaults(t *testing.T) {
	cfg := Default()
	if !cfg.AutoSessionDetect {
		t.Error("AutoSessionDetect should default to true")
	}
	if cfg.SessionDetectStartHour != 7 || cfg.SessionDetectEndHour != 22 {
		t.Errorf("detect window = %d-%d, want 7-22", cfg.SessionDetectStartHour, cfg.SessionDetectEndHour)
	}
	if cfg.SessionIdleEndMin != 30 {
		t.Errorf("SessionIdleEndMin = %d, want 30", cfg.SessionIdleEndMin)
	}
	if cfg.SessionIdleEndSec() != 1800 {
		t.Errorf("SessionIdleEndSec = %d, want 1800", cfg.SessionIdleEndSec())
	}
}

func TestMergeSessionDetection(t *testing.T) {
	cfg := mergeYAML(t, "auto_session_detect: false\nsession_detect_start_hour: 0\nsession_detect_end_hour: 23\nsession_idle_end_min: 45\n")

	if cfg.AutoSessionDetect {
		t.Error("auto_session_detect: false should be honoured")
	}
	if cfg.SessionDetectStartHour != 0 {
		t.Errorf("SessionDetectStartHour = %d, want 0", cfg.SessionDetectStartHour)
	}
	if cfg.SessionDetectEndHour != 23 {
		t.Errorf("SessionDetectEndHour = %d, want 23", cfg.SessionDetectEndHour)
	}
	if cfg.SessionIdleEndMin != 45 {
		t.Errorf("SessionIdleEndMin = %d, want 45", cfg.SessionIdleEndMin)
	}
}

func TestMergeSessionDetectionKeepsDefaultsWhenUnset(t *testing.T) {
	cfg := mergeYAML(t, "work_duration_min: 45\n")

	if !cfg.AutoSessionDetect {
		t.Error("AutoSessionDetect should stay true when the key is absent")
	}
	if cfg.SessionDetectStartHour != 7 || cfg.SessionDetectEndHour != 22 {
		t.Errorf("detect window = %d-%d, want the defaults 7-22", cfg.SessionDetectStartHour, cfg.SessionDetectEndHour)
	}
}

func TestTimerModeDefaultsToClassic(t *testing.T) {
	cfg := Default()
	if cfg.PomodoroEnabled() {
		t.Error("classic mode expected by default")
	}
	if cfg.EffectiveWorkMin() != 50 || cfg.EffectiveBreakMin(4) != 10 {
		t.Errorf("effective durations = %d/%d, want 50/10", cfg.EffectiveWorkMin(), cfg.EffectiveBreakMin(4))
	}
}

func TestPomodoroEffectiveDurations(t *testing.T) {
	cfg := mergeYAML(t, "timer_mode: pomodoro\npomodoro_work_min: 30\npomodoro_break_min: 6\npomodoro_long_break_min: 20\npomodoro_long_break_every: 3\n")

	if !cfg.PomodoroEnabled() {
		t.Fatal("pomodoro mode should be enabled")
	}
	if cfg.EffectiveWorkSec() != 30*60 {
		t.Errorf("EffectiveWorkSec = %d, want %d", cfg.EffectiveWorkSec(), 30*60)
	}
	for _, tc := range []struct {
		completed int
		wantMin   int
	}{
		{completed: 1, wantMin: 6},
		{completed: 2, wantMin: 6},
		{completed: 3, wantMin: 20},
		{completed: 6, wantMin: 20},
	} {
		if got := cfg.EffectiveBreakMin(tc.completed); got != tc.wantMin {
			t.Errorf("EffectiveBreakMin(%d) = %d, want %d", tc.completed, got, tc.wantMin)
		}
	}
	if cfg.LongBreakDue(0) {
		t.Error("LongBreakDue(0) should be false before any pomodoro completes")
	}
}

func TestApplyYAMLChangesRejectsInvalidTimerAndSessionValues(t *testing.T) {
	base := Default()
	cases := map[string]string{
		"unknown timer mode":     "timer_mode: sprint\n",
		"zero pomodoro work":     "pomodoro_work_min: 0\n",
		"negative long break":    "pomodoro_long_break_min: -5\n",
		"zero long break cycle":  "pomodoro_long_break_every: 0\n",
		"inverted detect window": "session_detect_start_hour: 20\nsession_detect_end_hour: 8\n",
		"zero idle end":          "session_idle_end_min: 0\n",
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			updated, err := ApplyYAMLChanges(base, []byte(change))
			if err == nil {
				t.Fatalf("ApplyYAMLChanges(%q) = %+v, want an error", change, updated)
			}
			if !reflect.DeepEqual(updated, base) {
				t.Error("rejected changes must leave the base config untouched")
			}
		})
	}
}

func TestApplyYAMLChangesAcceptsPomodoroMode(t *testing.T) {
	updated, err := ApplyYAMLChanges(Default(), []byte("timer_mode: pomodoro\n"))
	if err != nil {
		t.Fatalf("ApplyYAMLChanges: %v", err)
	}
	if !updated.PomodoroEnabled() {
		t.Errorf("TimerMode = %q, want pomodoro", updated.TimerMode)
	}
}
