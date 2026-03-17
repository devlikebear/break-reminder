package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
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

	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}

	merge(&cfg, &fileCfg)
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

func merge(dst, src *Config) {
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
	if src.MaxLogLines > 0 {
		dst.MaxLogLines = src.MaxLogLines
	}
	if src.CheckIntervalSec > 0 {
		dst.CheckIntervalSec = src.CheckIntervalSec
	}
	// Booleans: only override if YAML explicitly sets them
	// Since Go zero value for bool is false, we can't distinguish
	// "not set" from "set to false" with simple struct unmarshaling.
	// For now, keep defaults unless the source file explicitly has them.
}
