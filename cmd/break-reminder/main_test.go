package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestLoadConfigPreservesReturnedSettingsOnValidationError(t *testing.T) {
	origLoadConfig := loadConfig
	defer func() { loadConfig = origLoadConfig }()

	wantCfg := config.Default()
	wantCfg.Voice = "Samantha"
	wantCfg.NotificationsEnabled = false

	loadConfig = func() (config.Config, error) {
		return wantCfg, errors.New("work_end_minute must be between 0 and 59")
	}

	gotCfg, err := loadAppConfig()
	if err == nil {
		t.Fatal("loadAppConfig() error = nil, want validation error")
	}
	if !reflect.DeepEqual(gotCfg, wantCfg) {
		t.Fatalf("loadAppConfig() cfg = %+v, want %+v", gotCfg, wantCfg)
	}
}
