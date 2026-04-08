package dashboard

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
