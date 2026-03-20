package main

import (
	"errors"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

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

	ttsVoiceAvailable = func(engine, model, pythonCmd, voice string) bool {
		return true
	}
	var gotEngine, gotModel, gotPython, gotVoice, gotMessage string
	ttsSpeakAndWait = func(engine, model, pythonCmd, voice, message string) error {
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

	ttsVoiceAvailable = func(engine, model, pythonCmd, voice string) bool {
		return false
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, voice, message string) error {
		t.Fatal("ttsSpeakAndWait should not be called when voice is unavailable")
		return nil
	}

	if err := runTTSTest("안녕하세요"); err == nil {
		t.Fatal("expected runTTSTest() to fail when voice is unavailable")
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
	ttsVoiceAvailable = func(engine, model, pythonCmd, voice string) bool {
		return true
	}
	ttsSpeakAndWait = func(engine, model, pythonCmd, voice, message string) error {
		return wantErr
	}

	if err := runTTSTest("안녕하세요"); !errors.Is(err, wantErr) {
		t.Fatalf("runTTSTest() error = %v, want %v", err, wantErr)
	}
}
