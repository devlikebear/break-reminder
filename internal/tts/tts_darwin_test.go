//go:build darwin

package tts

import (
	"strings"
	"testing"
)

func TestDarwinSpeakerValidateGeminiNoAPIKey(t *testing.T) {
	s := NewSpeaker(engineGemini, "", "", "").(*DarwinSpeaker)
	err := s.validate("Zephyr")
	if err == nil {
		t.Fatal("expected error when GEMINI_API_KEY is empty")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatalf("error = %v, want mention of GEMINI_API_KEY", err)
	}
}

func TestDarwinSpeakerValidateGeminiUnknownVoice(t *testing.T) {
	s := NewSpeaker(engineGemini, "", "", "fake-key").(*DarwinSpeaker)
	err := s.validate("Yuna")
	if err == nil {
		t.Fatal("expected error for unknown voice")
	}
	if !strings.Contains(err.Error(), "not supported by Gemini") {
		t.Fatalf("error = %v, want 'not supported by Gemini'", err)
	}
}

func TestDarwinSpeakerAvailableGemini(t *testing.T) {
	// With valid voice + api key, Available should be true (afplay ships on macOS).
	s := NewSpeaker(engineGemini, "", "", "fake-key").(*DarwinSpeaker)
	if !s.Available("Zephyr") {
		t.Fatal("Available(Zephyr) with api key should be true on darwin")
	}
	// Missing api key -> unavailable.
	empty := NewSpeaker(engineGemini, "", "", "").(*DarwinSpeaker)
	if empty.Available("Zephyr") {
		t.Fatal("Available should be false without api key")
	}
	// Unknown voice -> unavailable.
	if s.Available("Yuna") {
		t.Fatal("Available(Yuna) should be false for gemini engine")
	}
}

func TestDarwinSpeakerGeminiNormalizesModel(t *testing.T) {
	s := NewSpeaker(engineGemini, "  ", "", "key").(*DarwinSpeaker)
	if s.model != defaultGeminiModel {
		t.Fatalf("model = %q, want default %q", s.model, defaultGeminiModel)
	}
	custom := NewSpeaker(engineGemini, "gemini-future-tts", "", "key").(*DarwinSpeaker)
	if custom.model != "gemini-future-tts" {
		t.Fatalf("model = %q, want override", custom.model)
	}
}
