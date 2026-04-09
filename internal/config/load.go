package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	configDir  = ".config/break-reminder"
	configFile = "config.yaml"
)

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir)
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), configFile)
}

// Load reads configuration from the YAML file, falling back to defaults.
func Load() (Config, error) {
	cfg := Default()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	// First unmarshal into a map to know which keys are explicitly set
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, err
	}

	// Then unmarshal into the struct for typed values
	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}

	merge(&cfg, &fileCfg, raw)
	if err := validateSchedule(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// EnsureConfigFile creates the default config file if it doesn't exist.
func EnsureConfigFile() error {
	path := ConfigPath()
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}

	cfg := Default()
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// Save writes the configuration to the YAML file.
func Save(cfg Config) error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}

	if err := validateSchedule(cfg); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0o644)
}

var validConfigKeys = map[string]struct{}{
	"work_duration_min":        {},
	"break_duration_min":       {},
	"idle_threshold_sec":       {},
	"natural_break_sec":        {},
	"work_days":                {},
	"work_start_hour":          {},
	"work_start_minute":        {},
	"work_end_hour":            {},
	"work_end_minute":          {},
	"voice":                    {},
	"tts_engine":               {},
	"tts_model":                {},
	"tts_python_cmd":           {},
	"ai_cli":                   {},
	"break_screen_mode":        {},
	"max_log_lines":            {},
	"check_interval_sec":       {},
	"notifications_enabled":    {},
	"tts_enabled":              {},
	"break_activities_enabled": {},
	"ai_enabled":               {},
}

// ApplyYAMLChanges merges YAML changes into an existing config and validates the result.
func ApplyYAMLChanges(base Config, changes []byte) (Config, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(changes, &raw); err != nil {
		return base, err
	}
	if len(raw) == 0 {
		return base, fmt.Errorf("no config changes found")
	}
	for key := range raw {
		if _, ok := validConfigKeys[key]; !ok {
			return base, fmt.Errorf("unknown config key %q", key)
		}
	}

	var patchCfg Config
	if err := yaml.Unmarshal(changes, &patchCfg); err != nil {
		return base, err
	}

	updated := base
	merge(&updated, &patchCfg, raw)
	if err := validateSchedule(updated); err != nil {
		return base, err
	}
	return updated, nil
}

func validateSchedule(cfg Config) error {
	if cfg.WorkStartHour < 0 || cfg.WorkStartHour > 23 {
		return fmt.Errorf("work_start_hour must be between 0 and 23")
	}
	if cfg.WorkEndHour < 0 || cfg.WorkEndHour > 23 {
		return fmt.Errorf("work_end_hour must be between 0 and 23")
	}
	if cfg.WorkStartMinute < 0 || cfg.WorkStartMinute > 59 {
		return fmt.Errorf("work_start_minute must be between 0 and 59")
	}
	if cfg.WorkEndMinute < 0 || cfg.WorkEndMinute > 59 {
		return fmt.Errorf("work_end_minute must be between 0 and 59")
	}

	workStartMinute := cfg.WorkStartHour*60 + cfg.WorkStartMinute
	workEndMinute := cfg.WorkEndHour*60 + cfg.WorkEndMinute
	if workEndMinute <= workStartMinute {
		return fmt.Errorf("work schedule must end after it starts")
	}

	return nil
}

func hasNonNilKey(raw map[string]any, key string) bool {
	value, ok := raw[key]
	return ok && value != nil
}

// merge applies values from src to dst. raw is the unmarshaled map used to
// detect which keys were explicitly set in the YAML file (needed for booleans and numeric zero values).
func merge(dst, src *Config, raw map[string]any) {
	if src.WorkDurationMin > 0 {
		dst.WorkDurationMin = src.WorkDurationMin
	}
	if src.BreakDurationMin > 0 {
		dst.BreakDurationMin = src.BreakDurationMin
	}
	if src.IdleThresholdSec > 0 {
		dst.IdleThresholdSec = src.IdleThresholdSec
	}
	if src.NaturalBreakSec > 0 {
		dst.NaturalBreakSec = src.NaturalBreakSec
	}
	if len(src.WorkDays) > 0 {
		dst.WorkDays = src.WorkDays
	}
	if hasNonNilKey(raw, "work_start_hour") {
		dst.WorkStartHour = src.WorkStartHour
	}
	if hasNonNilKey(raw, "work_start_minute") {
		dst.WorkStartMinute = src.WorkStartMinute
	}
	if hasNonNilKey(raw, "work_end_hour") {
		dst.WorkEndHour = src.WorkEndHour
	}
	if hasNonNilKey(raw, "work_end_minute") {
		dst.WorkEndMinute = src.WorkEndMinute
	}
	if src.Voice != "" {
		dst.Voice = src.Voice
	}
	if src.TTSEngine != "" {
		dst.TTSEngine = src.TTSEngine
	}
	if src.TTSModel != "" {
		dst.TTSModel = src.TTSModel
	}
	if src.TTSPythonCmd != "" {
		dst.TTSPythonCmd = src.TTSPythonCmd
	}
	if src.AICLI != "" {
		dst.AICLI = src.AICLI
	}
	if src.BreakScreenMode != "" {
		dst.BreakScreenMode = src.BreakScreenMode
	}
	if src.MaxLogLines > 0 {
		dst.MaxLogLines = src.MaxLogLines
	}
	if src.CheckIntervalSec > 0 {
		dst.CheckIntervalSec = src.CheckIntervalSec
	}

	// Booleans: only override if explicitly present in the YAML file
	if _, ok := raw["notifications_enabled"]; ok {
		dst.NotificationsEnabled = src.NotificationsEnabled
	}
	if _, ok := raw["tts_enabled"]; ok {
		dst.TTSEnabled = src.TTSEnabled
	}
	if _, ok := raw["break_activities_enabled"]; ok {
		dst.BreakActivitiesEnabled = src.BreakActivitiesEnabled
	}
	if _, ok := raw["ai_enabled"]; ok {
		dst.AIEnabled = src.AIEnabled
	}
}
