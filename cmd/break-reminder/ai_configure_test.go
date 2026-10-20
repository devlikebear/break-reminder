package main

import (
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestConfigurePromptNeverContainsSecrets(t *testing.T) {
	c := config.Default()
	c.TTSAPIKey = "AIza-should-not-leak"
	prompt := buildConfigurePrompt(c, "shorter breaks")
	if strings.Contains(prompt, "AIza-should-not-leak") {
		t.Fatalf("prompt leaks the key:\n%s", prompt)
	}
	if !strings.Contains(prompt, "shorter breaks") || !strings.Contains(prompt, "work_duration_min") {
		t.Fatalf("prompt lost its content:\n%s", prompt)
	}
}

func TestConfigureRejectsSecretChangesFromAI(t *testing.T) {
	if err := checkAIConfigChanges([]byte("work_duration_min: 25\n")); err != nil {
		t.Fatalf("plain change rejected: %v", err)
	}
	if err := checkAIConfigChanges([]byte("tts_api_key: <redacted>\nwork_duration_min: 25\n")); err == nil {
		t.Fatal("AI output touching tts_api_key must be rejected")
	}
}
