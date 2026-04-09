package dashboard

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestManualBreakResetsWarningBucket(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m := Model{
		cfg: config.Default(),
		state: state.State{
			Mode:                   "work",
			WorkSeconds:            1800,
			LastBreakWarningBucket: 4,
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	updated := updatedModel.(Model)

	if updated.state.Mode != "break" {
		t.Fatalf("Mode = %q, want break", updated.state.Mode)
	}
	if updated.state.WorkSeconds != 0 {
		t.Fatalf("WorkSeconds = %d, want 0", updated.state.WorkSeconds)
	}
	if updated.state.LastBreakWarningBucket != 0 {
		t.Fatalf("LastBreakWarningBucket = %d, want 0", updated.state.LastBreakWarningBucket)
	}
	if updated.state.BreakStart == 0 {
		t.Fatal("BreakStart was not set")
	}

	persisted, err := state.Load(filepath.Join(home, ".break-reminder-state"))
	if err != nil {
		t.Fatalf("Load saved state: %v", err)
	}
	if persisted.LastBreakWarningBucket != 0 {
		t.Fatalf("persisted LastBreakWarningBucket = %d, want 0", persisted.LastBreakWarningBucket)
	}
}

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
