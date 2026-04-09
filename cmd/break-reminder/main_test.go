package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestRootCommandAllowsRecoveryCommandsWithInvalidConfig(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("EDITOR", "true")
	configDir := filepath.Join(tmpHome, ".config", "break-reminder")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("work_end_minute: 75\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	for _, args := range [][]string{{"version"}, {"config", "path"}, {"config", "edit"}, {"pause"}, {"resume"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs(args)
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute(%v) error = %v, want nil", args, err)
			}
		})
	}
}

func TestRootCommandRejectsInvalidConfigForOperationalCommands(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	for _, args := range [][]string{{"status"}, {"config", "show"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs(args)
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("Execute(%v) error = nil, want validation error", args)
			}
			if err.Error() != "work_end_minute must be between 0 and 59" {
				t.Fatalf("Execute(%v) error = %q, want validation error", args, err.Error())
			}
		})
	}
}

func TestAICmdRejectsInvalidConfigBeforeCheckingAIFeatures(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.AIEnabled = true
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	cmd := newAICmd()
	cmd.SetArgs([]string{"suggest"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if err.Error() != "work_end_minute must be between 0 and 59" {
		t.Fatalf("Execute() error = %q, want validation error", err.Error())
	}
}

func TestSetAppConfigPreservesCurrentConfigOnValidationError(t *testing.T) {
	origLoadConfig := loadConfig
	origCfg := cfg
	defer func() {
		loadConfig = origLoadConfig
		cfg = origCfg
	}()

	currentCfg := config.Default()
	currentCfg.WorkStartMinute = 15
	cfg = currentCfg

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	err := setAppConfig()
	if err == nil {
		t.Fatal("setAppConfig() error = nil, want validation error")
	}
	if !reflect.DeepEqual(cfg, currentCfg) {
		t.Fatalf("cfg mutated on validation error: got %+v want %+v", cfg, currentCfg)
	}
}
