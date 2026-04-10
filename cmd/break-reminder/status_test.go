package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestStatusShowsPausedBreakState(t *testing.T) {
	origNowFunc := nowFunc
	origCfg := cfg
	defer func() {
		nowFunc = origNowFunc
		cfg = origCfg
	}()

	cfg = config.Default()
	nowFunc = func() time.Time {
		return time.Unix(1_700_000_660, 0)
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	statePath := filepath.Join(tmpHome, ".break-reminder-state")
	if err := state.Save(statePath, state.State{
		Mode:           "break",
		BreakStart:     1_700_000_000,
		Paused:         true,
		PausedAt:       1_700_000_360,
		LastUpdateDate: "2025-01-15",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cmd := newStatusCmd()
	cmd.SetArgs(nil)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status Execute() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Mode: paused (break)",
		"Paused for: 5m0s",
		"Break elapsed: 6min / 10min",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status output missing %q\nfull output:\n%s", want, got)
		}
	}
}
