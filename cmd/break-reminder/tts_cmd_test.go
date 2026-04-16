package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func findTTSSubcommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	root := newTTSCmd()
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func TestSetAPIKeyFromArg(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	if err := runSetAPIKey([]string{"AIzaSy-test-key"}); err != nil {
		t.Fatalf("runSetAPIKey() error = %v", err)
	}
	if cfg.TTSAPIKey != "AIzaSy-test-key" {
		t.Fatalf("in-memory cfg.TTSAPIKey = %q, want AIzaSy-test-key", cfg.TTSAPIKey)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if loaded.TTSAPIKey != "AIzaSy-test-key" {
		t.Fatalf("loaded cfg.TTSAPIKey = %q, want AIzaSy-test-key", loaded.TTSAPIKey)
	}

	info, err := os.Stat(config.ConfigPath())
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config perm = %o, want 0600", perm)
	}
}

func TestSetAPIKeyFromStdin(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	origStdin := stdinForSetAPIKey
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
		stdinForSetAPIKey = origStdin
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return false }
	stdinForSetAPIKey = strings.NewReader("piped-key-123\n")

	if err := runSetAPIKey(nil); err != nil {
		t.Fatalf("runSetAPIKey() error = %v", err)
	}
	if cfg.TTSAPIKey != "piped-key-123" {
		t.Fatalf("cfg.TTSAPIKey = %q, want piped-key-123", cfg.TTSAPIKey)
	}
}

func TestSetAPIKeyTrimsWhitespace(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	if err := runSetAPIKey([]string{"  padded-key  "}); err != nil {
		t.Fatalf("runSetAPIKey() error = %v", err)
	}
	if cfg.TTSAPIKey != "padded-key" {
		t.Fatalf("cfg.TTSAPIKey = %q, want padded-key", cfg.TTSAPIKey)
	}
}

func TestSetAPIKeyRejectsEmpty(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	err := runSetAPIKey([]string{"   "})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("err = %v, want 'must not be empty'", err)
	}
}

func TestSetAPIKeyRejectsNewlineInArg(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	err := runSetAPIKey([]string{"line1\nline2"})
	if err == nil {
		t.Fatal("expected error for multi-line key")
	}
	if !strings.Contains(err.Error(), "newline") {
		t.Fatalf("err = %v, want mention of newline", err)
	}
}

func TestSetAPIKeyRefusesTTYWithoutArg(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	err := runSetAPIKey(nil)
	if err == nil {
		t.Fatal("expected error when tty and no arg")
	}
	if !strings.Contains(err.Error(), "pass key as argument") {
		t.Fatalf("err = %v, want 'pass key as argument' hint", err)
	}
}

func TestSetAPIKeyOutputDoesNotLeakKey(t *testing.T) {
	withTempConfigDir(t)
	origCfg := cfg
	origTerm := stdinIsTerminal
	defer func() {
		cfg = origCfg
		stdinIsTerminal = origTerm
	}()
	cfg = config.Default()
	stdinIsTerminal = func() bool { return true }

	// Capture stdout
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	secret := "super-secret-AIzaSy"
	if err := runSetAPIKey([]string{secret}); err != nil {
		t.Fatalf("runSetAPIKey() error = %v", err)
	}
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("stdout leaked secret key: %q", buf.String())
	}
}

func TestConfigPathResolvesUnderTempHome(t *testing.T) {
	dir := withTempConfigDir(t)
	want := filepath.Join(dir, ".config", "break-reminder", "config.yaml")
	if got := config.ConfigPath(); got != want {
		t.Fatalf("ConfigPath() = %q, want %q (t.Setenv HOME honored?)", got, want)
	}
}

func TestTTSInstallRejectsGemini(t *testing.T) {
	origCfg := cfg
	defer func() { cfg = origCfg }()
	cfg = config.Default()

	sub := findTTSSubcommand(t, "install")
	var out bytes.Buffer
	sub.SetOut(&out)
	sub.SetErr(&out)
	err := sub.RunE(sub, []string{"gemini"})
	if err == nil {
		t.Fatal("expected error when installing gemini, got nil")
	}
	if !strings.Contains(err.Error(), "gemini engine requires no install") {
		t.Fatalf("error = %v, want 'gemini engine requires no install' hint", err)
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatalf("error = %v, want GEMINI_API_KEY hint", err)
	}
}

func TestRunTTSTestSpeaksConfiguredMessage(t *testing.T) {
	origCfg := cfg
	origAvailable := ttsVoiceAvailable
	origSpeak := ttsSpeakAndWait
	defer func() {
		cfg = origCfg
		ttsVoiceAvailable = origAvailable
		ttsSpeakAndWait = origSpeak
	}()

	cfg = config.Default()
	cfg.TTSEngine = "kittentts"
	cfg.Voice = "Jasper"

	ttsVoiceAvailable = func(engine, model, pythonCmd, apiKey, voice string) bool {
		return true
	}
	var gotEngine, gotModel, gotPython, gotVoice, gotMessage string
	ttsSpeakAndWait = func(engine, model, pythonCmd, apiKey, voice, message string) error {
		gotEngine = engine
		gotModel = model
		gotPython = pythonCmd
		gotVoice = voice
		gotMessage = message
		return nil
	}

	if err := runTTSTest("안녕하세요"); err != nil {
		t.Fatalf("runTTSTest() error = %v", err)
	}
	if gotEngine != "kittentts" {
		t.Fatalf("engine = %q, want kittentts", gotEngine)
	}
	if gotVoice != "Jasper" {
		t.Fatalf("voice = %q, want Jasper", gotVoice)
	}
	if gotMessage != "안녕하세요" {
		t.Fatalf("message = %q, want 안녕하세요", gotMessage)
	}
	if gotModel != cfg.TTSModel {
		t.Fatalf("model = %q, want %q", gotModel, cfg.TTSModel)
	}
	if gotPython != cfg.TTSPythonCmd {
		t.Fatalf("pythonCmd = %q, want %q", gotPython, cfg.TTSPythonCmd)
	}
}

func TestRunTTSTestFailsWhenVoiceUnavailable(t *testing.T) {
	origCfg := cfg
	origAvailable := ttsVoiceAvailable
	origSpeak := ttsSpeakAndWait
	defer func() {
		cfg = origCfg
		ttsVoiceAvailable = origAvailable
		ttsSpeakAndWait = origSpeak
	}()

	cfg = config.Default()
	cfg.TTSEngine = "kittentts"
	cfg.Voice = "Jasper"

	ttsVoiceAvailable = func(engine, model, pythonCmd, apiKey, voice string) bool {
		return false
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, apiKey, voice, message string) error {
		t.Fatal("ttsSpeakAndWait should not be called when voice is unavailable")
		return nil
	}

	if err := runTTSTest("안녕하세요"); err == nil {
		t.Fatal("expected runTTSTest() to fail when voice is unavailable")
	}
}

func TestRunTTSTestForwardsGeminiAPIKey(t *testing.T) {
	origCfg := cfg
	origAvailable := ttsVoiceAvailable
	origSpeak := ttsSpeakAndWait
	defer func() {
		cfg = origCfg
		ttsVoiceAvailable = origAvailable
		ttsSpeakAndWait = origSpeak
	}()

	t.Setenv("GEMINI_API_KEY", "env-fallback-key")

	cfg = config.Default()
	cfg.TTSEngine = "gemini"
	cfg.TTSModel = "gemini-3.1-flash-tts-preview"
	cfg.Voice = "Zephyr"
	cfg.TTSAPIKey = "" // force env fallback

	var gotEngine, gotAPIKey, gotVoice string
	ttsVoiceAvailable = func(engine, model, pythonCmd, apiKey, voice string) bool {
		gotAPIKey = apiKey
		return true
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, apiKey, voice, message string) error {
		gotEngine = engine
		gotVoice = voice
		return nil
	}

	if err := runTTSTest("hello"); err != nil {
		t.Fatalf("runTTSTest() error = %v", err)
	}
	if gotEngine != "gemini" {
		t.Fatalf("engine = %q, want gemini", gotEngine)
	}
	if gotVoice != "Zephyr" {
		t.Fatalf("voice = %q, want Zephyr", gotVoice)
	}
	if gotAPIKey != "env-fallback-key" {
		t.Fatalf("apiKey = %q, want env-fallback-key (env fallback)", gotAPIKey)
	}
}

func TestRunTTSTestPrefersConfigAPIKeyOverEnv(t *testing.T) {
	origCfg := cfg
	origAvailable := ttsVoiceAvailable
	origSpeak := ttsSpeakAndWait
	defer func() {
		cfg = origCfg
		ttsVoiceAvailable = origAvailable
		ttsSpeakAndWait = origSpeak
	}()

	t.Setenv("GEMINI_API_KEY", "env-key")

	cfg = config.Default()
	cfg.TTSEngine = "gemini"
	cfg.Voice = "Zephyr"
	cfg.TTSAPIKey = "config-key"

	var gotAPIKey string
	ttsVoiceAvailable = func(engine, model, pythonCmd, apiKey, voice string) bool {
		gotAPIKey = apiKey
		return true
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, apiKey, voice, message string) error {
		return nil
	}

	if err := runTTSTest("hello"); err != nil {
		t.Fatalf("runTTSTest() error = %v", err)
	}
	if gotAPIKey != "config-key" {
		t.Fatalf("apiKey = %q, want config-key (config wins over env)", gotAPIKey)
	}
}

func TestRunTTSTestPropagatesSpeakError(t *testing.T) {
	origCfg := cfg
	origAvailable := ttsVoiceAvailable
	origSpeak := ttsSpeakAndWait
	defer func() {
		cfg = origCfg
		ttsVoiceAvailable = origAvailable
		ttsSpeakAndWait = origSpeak
	}()

	cfg = config.Default()
	cfg.Voice = "Yuna"

	wantErr := errors.New("speak failed")
	ttsVoiceAvailable = func(engine, model, pythonCmd, apiKey, voice string) bool {
		return true
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, apiKey, voice, message string) error {
		return wantErr
	}

	if err := runTTSTest("안녕하세요"); !errors.Is(err, wantErr) {
		t.Fatalf("runTTSTest() error = %v, want %v", err, wantErr)
	}
}
