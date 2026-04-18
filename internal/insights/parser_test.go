package insights

import (
	"strings"
	"testing"
)

func TestParseResponseValid(t *testing.T) {
	response := `{
      "daily_report": "오늘 4시간 작업했어요.",
      "patterns": [
        {"type": "warning", "title": "슬럼프", "description": "D1", "suggestion": "S1"},
        {"type": "info", "title": "골든타임", "description": "D2", "suggestion": "S2"}
      ]
    }`

	report, err := ParseResponse(response)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if report.DailyReport != "오늘 4시간 작업했어요." {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
	if len(report.Patterns) != 2 {
		t.Errorf("Patterns count = %d, want 2", len(report.Patterns))
	}
	if report.Patterns[0].Type != "warning" {
		t.Errorf("Patterns[0].Type = %q, want warning", report.Patterns[0].Type)
	}
}

func TestParseResponseStripsCodeFences(t *testing.T) {
	response := "```json\n" + `{"daily_report":"ok","patterns":[]}` + "\n```"
	report, err := ParseResponse(response)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if report.DailyReport != "ok" {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
}

func TestParseResponseInvalidJSON(t *testing.T) {
	_, err := ParseResponse("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}
