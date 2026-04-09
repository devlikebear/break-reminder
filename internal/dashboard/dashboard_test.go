package dashboard

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestUpdateLogsInvalidConfigReloadOnlyOncePerError(t *testing.T) {
	origLoadConfig := loadConfig
	origLogger := log.Logger
	defer func() {
		loadConfig = origLoadConfig
		log.Logger = origLogger
	}()

	loadConfig = func() (config.Config, error) {
		return config.Default(), errors.New("work_end_minute must be between 0 and 59")
	}

	var logs bytes.Buffer
	log.Logger = zerolog.New(&logs)

	model := New(config.Default())
	updated, _ := model.Update(tickMsg(time.Now()))
	model = updated.(Model)
	updated, _ = model.Update(tickMsg(time.Now().Add(time.Second)))
	model = updated.(Model)

	if got := strings.Count(logs.String(), "Ignoring invalid config reload"); got != 1 {
		t.Fatalf("warning count = %d, want 1; logs=%q", got, logs.String())
	}
	if model.lastConfigError != "work_end_minute must be between 0 and 59" {
		t.Fatalf("lastConfigError = %q, want validation error", model.lastConfigError)
	}
}

func TestUpdateLogsInvalidConfigReloadAgainAfterRecovery(t *testing.T) {
	origLoadConfig := loadConfig
	origLogger := log.Logger
	defer func() {
		loadConfig = origLoadConfig
		log.Logger = origLogger
	}()

	calls := 0
	loadConfig = func() (config.Config, error) {
		calls++
		switch calls {
		case 1:
			return config.Default(), errors.New("work_end_minute must be between 0 and 59")
		case 2:
			cfg := config.Default()
			cfg.WorkStartMinute = 30
			return cfg, nil
		default:
			return config.Default(), errors.New("work_end_minute must be between 0 and 59")
		}
	}

	var logs bytes.Buffer
	log.Logger = zerolog.New(&logs)

	model := New(config.Default())
	updated, _ := model.Update(tickMsg(time.Now()))
	model = updated.(Model)
	updated, _ = model.Update(tickMsg(time.Now().Add(time.Second)))
	model = updated.(Model)
	updated, _ = model.Update(tickMsg(time.Now().Add(2 * time.Second)))
	model = updated.(Model)

	if got := strings.Count(logs.String(), "Ignoring invalid config reload"); got != 2 {
		t.Fatalf("warning count = %d, want 2; logs=%q", got, logs.String())
	}
	if model.lastConfigError != "work_end_minute must be between 0 and 59" {
		t.Fatalf("lastConfigError = %q, want validation error", model.lastConfigError)
	}
	if model.cfg.WorkStartMinute != 30 {
		t.Fatalf("WorkStartMinute = %d, want 30 from recovered config", model.cfg.WorkStartMinute)
	}
}
