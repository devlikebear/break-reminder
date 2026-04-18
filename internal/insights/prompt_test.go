package insights

import (
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/ai"
)

func TestBuildPromptIncludesHistory(t *testing.T) {
	history := []ai.DailySummary{
		{Date: "2026-04-17", WorkMin: 280, BreakMin: 60, Sessions: 7, Activities: 3},
		{Date: "2026-04-16", WorkMin: 200, BreakMin: 40, Sessions: 4, Activities: 2},
	}

	prompt := BuildPrompt(history)
	if !strings.Contains(prompt, "2026-04-17") {
		t.Error("prompt missing today's date")
	}
	if !strings.Contains(prompt, "daily_report") {
		t.Error("prompt should request daily_report field")
	}
	if !strings.Contains(prompt, "patterns") {
		t.Error("prompt should request patterns field")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should request JSON format")
	}
}

func TestBuildPromptEmptyHistory(t *testing.T) {
	prompt := BuildPrompt(nil)
	if prompt == "" {
		t.Error("empty prompt for no history")
	}
	if !strings.Contains(prompt, "[]") {
		t.Error("should embed empty history array")
	}
}
