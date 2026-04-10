package breakscreen

import (
	"os"
	"path/filepath"
	"testing"
)

func stubHelperLookup(t *testing.T, exePath, homeDir string) {
	t.Helper()

	origExecutablePath := executablePath
	origUserHomeDir := userHomeDir
	executablePath = func() (string, error) { return exePath, nil }
	userHomeDir = func() (string, error) { return homeDir, nil }
	t.Cleanup(func() {
		executablePath = origExecutablePath
		userHomeDir = origUserHomeDir
	})
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func TestFindHelperUsesExecutableDir(t *testing.T) {
	root := t.TempDir()
	exePath := filepath.Join(root, "app", "break-reminder")
	helperPath := filepath.Join(root, "app", "test-helper")
	writeExecutable(t, helperPath)
	stubHelperLookup(t, exePath, filepath.Join(root, "home"))

	result := FindHelper("test-helper")
	if result != helperPath {
		t.Fatalf("FindHelper() = %q, want %q", result, helperPath)
	}
}

func TestFindHelperUsesInstalledLocalBin(t *testing.T) {
	root := t.TempDir()
	homeDir := filepath.Join(root, "home")
	helperPath := filepath.Join(homeDir, ".local", "bin", "test-helper")
	writeExecutable(t, helperPath)
	stubHelperLookup(t, filepath.Join(root, "app", "break-reminder"), homeDir)

	result := FindHelper("test-helper")
	if result != helperPath {
		t.Fatalf("FindHelper() = %q, want %q", result, helperPath)
	}
}

func TestFindHelperDoesNotTrustCurrentWorkingDirectoryBin(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	if err := os.MkdirAll(filepath.Join(cwd, "bin"), 0o755); err != nil {
		t.Fatalf("MkdirAll(cwd/bin): %v", err)
	}
	maliciousPath := filepath.Join(cwd, "bin", "test-helper")
	writeExecutable(t, maliciousPath)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("Chdir(%q): %v", cwd, err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	stubHelperLookup(t, filepath.Join(root, "app", "break-reminder"), filepath.Join(root, "home"))

	result := FindHelper("test-helper")
	if result != "" {
		t.Fatalf("FindHelper() = %q, want empty result for cwd bin helper", result)
	}
}

func TestFindHelperDoesNotTrustPATH(t *testing.T) {
	root := t.TempDir()
	pathDir := filepath.Join(root, "path")
	maliciousPath := filepath.Join(pathDir, "test-helper")
	writeExecutable(t, maliciousPath)
	t.Setenv("PATH", pathDir)
	stubHelperLookup(t, filepath.Join(root, "app", "break-reminder"), filepath.Join(root, "home"))

	result := FindHelper("test-helper")
	if result != "" {
		t.Fatalf("FindHelper() = %q, want empty result for PATH helper", result)
	}
}

func TestFindHelperNotFound(t *testing.T) {
	root := t.TempDir()
	stubHelperLookup(t, filepath.Join(root, "app", "break-reminder"), filepath.Join(root, "home"))

	result := FindHelper("nonexistent-helper-xyz-12345")
	if result != "" {
		t.Errorf("FindHelper should return empty for missing binary, got %q", result)
	}
}
