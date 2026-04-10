package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestStatusShowsPausedState(t *testing.T) {
	origCfg := cfg
	defer func() { cfg = origCfg }()

	cfg = config.Default()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	now := time.Now()
	if err := state.Save(state.DefaultStatePath(), state.State{
		Mode:           "break",
		Paused:         true,
		PausedAt:       now.Add(-5 * time.Minute).Unix(),
		LastUpdateDate: now.Format("2006-01-02"),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cmd := newStatusCmd()
	cmd.SetArgs(nil)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "Mode: paused (break)") {
		t.Fatalf("status output = %q, want paused mode label", text)
	}
	if !strings.Contains(text, "Paused for:") {
		t.Fatalf("status output = %q, want paused duration", text)
	}
}
