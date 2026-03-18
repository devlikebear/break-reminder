package breakscreen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindHelperInBinDir(t *testing.T) {
	// Create a temp binary in bin/ relative to cwd
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(dir)

	binDir := filepath.Join(dir, "bin")
	_ = os.MkdirAll(binDir, 0o755)
	helperPath := filepath.Join(binDir, "test-helper")
	_ = os.WriteFile(helperPath, []byte("#!/bin/sh\n"), 0o755)

	result := FindHelper("test-helper")
	if result == "" {
		t.Error("FindHelper should find binary in bin/ directory")
	}
}

func TestFindHelperNotFound(t *testing.T) {
	result := FindHelper("nonexistent-helper-xyz-12345")
	if result != "" {
		t.Errorf("FindHelper should return empty for missing binary, got %q", result)
	}
}
