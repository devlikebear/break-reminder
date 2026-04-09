package main

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestRootCommandRejectsInvalidConfigAtStartup(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	loadConfig = func() (config.Config, error) {
		invalidCfg := config.Default()
		invalidCfg.WorkEndMinute = 75
		return invalidCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"version"})
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
