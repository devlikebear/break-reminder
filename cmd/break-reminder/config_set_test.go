package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

func setupConfigTestHome(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := config.EnsureConfigFile(); err != nil {
		t.Fatalf("EnsureConfigFile: %v", err)
	}
	return tmpHome
}

func runRootCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	origLoadConfig := loadConfig
	origCfg := cfg
	t.Cleanup(func() {
		loadConfig = origLoadConfig
		cfg = origCfg
	})
	loadConfig = config.Load
	root := newRootCmd()
	root.SetArgs(args)
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	err := root.Execute()
	return out.String(), err
}

func TestConfigSetSingleIntKey(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // This test asserts the English diagnostic contract.
	setupConfigTestHome(t)

	out, err := runRootCmd(t, "config", "set", "work_duration_min=45")
	if err != nil {
		t.Fatalf("config set: %v (output=%q)", err, out)
	}
	if !strings.Contains(out, "Configuration updated") {
		t.Fatalf("output = %q, want confirmation", out)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.WorkDurationMin != 45 {
		t.Fatalf("WorkDurationMin = %d, want 45", loaded.WorkDurationMin)
	}
}

func TestConfigSetMultipleKeys(t *testing.T) {
	setupConfigTestHome(t)

	_, err := runRootCmd(t, "config", "set", "work_duration_min=45", "break_duration_min=15")
	if err != nil {
		t.Fatalf("config set multi: %v", err)
	}

	loaded, _ := config.Load()
	if loaded.WorkDurationMin != 45 || loaded.BreakDurationMin != 15 {
		t.Fatalf("got work=%d break=%d, want 45/15", loaded.WorkDurationMin, loaded.BreakDurationMin)
	}
}

func TestConfigSetBoolKey(t *testing.T) {
	setupConfigTestHome(t)

	_, err := runRootCmd(t, "config", "set", "notifications_enabled=false")
	if err != nil {
		t.Fatalf("config set bool: %v", err)
	}

	loaded, _ := config.Load()
	if loaded.NotificationsEnabled {
		t.Fatal("NotificationsEnabled should be false")
	}
}

func TestConfigSetRejectsInvalidScheduleAtomically(t *testing.T) {
	home := setupConfigTestHome(t)
	configPath := filepath.Join(home, ".config", "break-reminder", "config.yaml")
	originalBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile pre: %v", err)
	}

	_, err = runRootCmd(t, "config", "set", "work_duration_min=99", "work_start_hour=25")
	if err == nil {
		t.Fatal("expected error from invalid work_start_hour")
	}

	afterBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile post: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Fatal("config file was modified despite validation failure (not atomic)")
	}
}

func TestConfigSetRejectsUnknownKey(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en") // This test asserts the English diagnostic contract.
	setupConfigTestHome(t)

	_, err := runRootCmd(t, "config", "set", "unknown_key=foo")
	if err == nil {
		t.Fatal("expected unknown-key error")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Fatalf("error = %v, want unknown-key", err)
	}
}

func TestConfigSetMalformedArg(t *testing.T) {
	setupConfigTestHome(t)

	_, err := runRootCmd(t, "config", "set", "no_equals_sign")
	if err == nil {
		t.Fatal("expected error for malformed arg")
	}
}

func TestConfigGetReturnsValue(t *testing.T) {
	setupConfigTestHome(t)

	if _, err := runRootCmd(t, "config", "set", "work_duration_min=42"); err != nil {
		t.Fatalf("config set precondition: %v", err)
	}

	out, err := runRootCmd(t, "config", "get", "work_duration_min")
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(out, "42") {
		t.Fatalf("output = %q, want 42", out)
	}
}

func TestConfigSetRoutesSecretToKeychain(t *testing.T) {
	home := setupConfigTestHome(t)
	orig := secrets.Default
	store := secrets.NewMemory()
	secrets.Default = store
	t.Cleanup(func() { secrets.Default = orig })

	if _, err := runRootCmd(t, "config", "set", "tts_api_key=AIza-from-set", "work_duration_min=40"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	if v, _ := store.Get(secrets.TTSAPIKey); v != "AIza-from-set" {
		t.Fatalf("stored = %q", v)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".config", "break-reminder", "config.yaml"))
	if strings.Contains(string(data), "AIza-from-set") {
		t.Fatalf("secret written to config file:\n%s", data)
	}
	loaded, _ := config.Load()
	if loaded.WorkDurationMin != 40 {
		t.Fatalf("other keys should still apply, work=%d", loaded.WorkDurationMin)
	}
}

func TestLegacySecretMigratesOutOfConfigAndIsMasked(t *testing.T) {
	home := setupConfigTestHome(t)
	orig := secrets.Default
	store := secrets.NewMemory()
	secrets.Default = store
	t.Cleanup(func() { secrets.Default = orig })

	path := filepath.Join(home, ".config", "break-reminder", "config.yaml")
	if err := os.WriteFile(path, []byte("tts_api_key: AIza-legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRootCmd(t, "config", "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if strings.Contains(out, "AIza-legacy") {
		t.Fatalf("show leaked the key: %q", out)
	}
	if v, _ := store.Get(secrets.TTSAPIKey); v != "AIza-legacy" {
		t.Fatalf("migration stored %q", v)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "AIza-legacy") {
		t.Fatalf("key left in config.yaml:\n%s", data)
	}
	info, _ := os.Stat(path)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 600", perm)
	}
}

func TestConfigShowAndGetMaskUnmigratedSecret(t *testing.T) {
	setupConfigTestHome(t)
	orig := secrets.Default
	secrets.Default = failingStore{}
	t.Cleanup(func() { secrets.Default = orig })

	path := config.ConfigPath()
	if err := os.WriteFile(path, []byte("tts_api_key: AIza-stuck\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Migration fails (keychain unavailable), so the key stays in the file;
	// it still must never be printed.
	for _, args := range [][]string{{"config", "show"}, {"config", "get", "tts_api_key"}} {
		out, err := runRootCmd(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if strings.Contains(out, "AIza-stuck") || !strings.Contains(out, config.RedactedValue) {
			t.Fatalf("%v output = %q", args, out)
		}
	}
}

type failingStore struct{}

func (failingStore) Get(string) (string, error) { return "", secrets.ErrNotFound }
func (failingStore) Set(string, string) error   { return os.ErrPermission }
func (failingStore) Delete(string) error        { return nil }
