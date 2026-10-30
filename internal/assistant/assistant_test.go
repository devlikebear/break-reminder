package assistant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/ai"
)

func TestMain(m *testing.M) {
	os.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	os.Exit(m.Run())
}

func facts() Facts {
	return Facts{
		Date: "2026-10-27", YesterdayWorkMin: 372, YesterdayBreakMin: 48,
		TodayEvents: 3, NextEventTitle: "Design review", NextEventInMin: 40,
		FocusMinutes: 40, FocusPomodoros: 1, TodayWorkMin: 310, TodayBreakMin: 35, TodayPomodoros: 4,
		Week: []ai.DailySummary{{Date: "2026-10-26", WorkMin: 372, BreakMin: 48}},
	}
}

func TestBriefingPromptCarriesNumbersNotRawData(t *testing.T) {
	p := BriefingPrompt(facts())
	for _, want := range []string{"372", "48", "3", "40", "English", "3 sentences"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q:\n%s", want, p)
		}
	}
	if !strings.Contains(p, "Design review") {
		t.Error("shared title missing")
	}
	f := facts()
	f.NextEventTitle = ""
	if strings.Contains(BriefingPrompt(f), "Design review") {
		t.Error("unshared title leaked")
	}
}

func TestTemplatesWorkWithoutAI(t *testing.T) {
	b := TemplateBriefing(facts())
	if !strings.Contains(b, "6h 12m") || !strings.Contains(b, "40") {
		t.Fatalf("briefing = %q", b)
	}
	r := TemplateRetro(facts())
	if !strings.Contains(r, "5h 10m") || !strings.Contains(r, "4") {
		t.Fatalf("retro = %q", r)
	}
	empty := TemplateBriefing(Facts{Date: "2026-10-27"})
	if empty == "" || strings.Contains(empty, "Yesterday") {
		t.Fatalf("no history: %q", empty)
	}
}

func TestParseRetroReadsJSONOrPlainText(t *testing.T) {
	retro, report, err := ParseRetro("```json\n{\"retro\":\"Solid day.\",\"daily_report\":\"5h of work.\",\"patterns\":[{\"type\":\"info\",\"title\":\"Peak\",\"description\":\"Mornings\",\"suggestion\":\"Guard them\"}]}\n```")
	if err != nil || retro != "Solid day." || report == nil || report.DailyReport != "5h of work." || len(report.Patterns) != 1 {
		t.Fatalf("json -> %q %+v %v", retro, report, err)
	}
	retro, report, err = ParseRetro("Just a sentence.")
	if err != nil || retro != "Just a sentence." || report != nil {
		t.Fatalf("plain -> %q %+v %v", retro, report, err)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "assistant")}
	d, err := s.Load("2026-10-27")
	if err != nil || d.Date != "2026-10-27" || d.Briefing != "" {
		t.Fatalf("empty load = %+v %v", d, err)
	}
	d.Briefing, d.BriefingSource = "Hello", SourceAI
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load("2026-10-27")
	if got.Briefing != "Hello" || got.BriefingSource != SourceAI {
		t.Fatalf("got %+v", got)
	}
	if info, _ := os.Stat(filepath.Join(s.Dir, "2026-10-27.json")); info.Mode().Perm() != 0o600 {
		t.Fatal("day file must be 0600")
	}
}
