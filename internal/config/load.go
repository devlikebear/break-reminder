package config

import (
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

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0o644)
}

// merge applies values from src to dst. raw is the unmarshaled map used to
// detect which keys were explicitly set in the YAML file (needed for booleans).
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
	if src.WorkStartHour > 0 {
		dst.WorkStartHour = src.WorkStartHour
	}
	if src.WorkEndHour > 0 {
		dst.WorkEndHour = src.WorkEndHour
	}
	if src.Voice != "" {
		dst.Voice = src.Voice
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
