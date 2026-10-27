// Package assistant writes the daily briefing and retrospective. The numbers
// are computed here; a model (when the user consented) only turns them into
// sentences, and a template does so otherwise.
package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/insights"
	"github.com/devlikebear/break-reminder/internal/reminder"
)

// Where a text came from.
const (
	SourceAI       = "ai"
	SourceTemplate = "template"
)

// Day is one day's briefing and retrospective.
type Day struct {
	Date           string `json:"date"`
	Briefing       string `json:"briefing,omitempty"`
	BriefingSource string `json:"briefing_source,omitempty"`
	BriefingAt     int64  `json:"briefing_at,omitempty"`
	Retro          string `json:"retro,omitempty"`
	RetroSource    string `json:"retro_source,omitempty"`
	RetroAt        int64  `json:"retro_at,omitempty"`
	Dismissed      bool   `json:"dismissed,omitempty"`
}

// Facts are the numbers a briefing or retro is written from.
type Facts struct {
	Date              string
	YesterdayWorkMin  int
	YesterdayBreakMin int
	TodayEvents       int
	NextEventTitle    string // empty unless the user shares titles
	NextEventInMin    int    // -1 or 0 when there is none
	FocusMinutes      int
	FocusPomodoros    int
	TodayWorkMin      int
	TodayBreakMin     int
	TodayPomodoros    int
	Week              []ai.DailySummary // recent finished days, for the retro
}

// SystemPrompt frames both texts.
func SystemPrompt() string {
	return "You write short notes for Break Reminder, a macOS app that keeps a healthy work/break rhythm. " +
		"Use only the numbers given; never invent events or figures. Reply in " + i18n.ResponseLanguage() + "."
}

// BriefingPrompt asks for the morning briefing.
func BriefingPrompt(f Facts) string {
	lines := []string{
		fmt.Sprintf("Write today's (%s) start-of-day briefing: at most 3 sentences, numbers first, and end with one concrete suggestion for today.", f.Date),
		"",
		fmt.Sprintf("- yesterday: worked %d min, took %d min of breaks", f.YesterdayWorkMin, f.YesterdayBreakMin),
		fmt.Sprintf("- calendar events today: %d", f.TodayEvents),
	}
	if f.NextEventInMin > 0 {
		if f.NextEventTitle != "" {
			lines = append(lines, fmt.Sprintf("- next event: %q in %d min", f.NextEventTitle, f.NextEventInMin))
		} else {
			lines = append(lines, fmt.Sprintf("- next event in %d min", f.NextEventInMin))
		}
	}
	lines = append(lines,
		fmt.Sprintf("- uninterrupted focus time now: %d min (%d pomodoros)", f.FocusMinutes, f.FocusPomodoros),
		"", "Plain text only, no markdown. Write in "+i18n.ResponseLanguage()+".")
	return strings.Join(lines, "\n")
}

// RetroPrompt asks for the end-of-session retrospective and, in the same
// call, the insights report (one AI call covers both).
func RetroPrompt(f Facts) string {
	week, _ := json.Marshal(f.Week)
	return fmt.Sprintf(`Write the end-of-day retrospective for %s and analyze the recent history.

Today: worked %d min, %d min of breaks, %d pomodoros.
Recent finished days: %s

Return JSON only, no markdown fences:
{"retro":"at most 3 sentences on today, numbers first, one thing to keep or change tomorrow",
 "daily_report":"2-3 sentence summary of the recent days",
 "patterns":[{"type":"warning|positive|info","title":"...","description":"...","suggestion":"..."}]}
Give 2-3 patterns. Keep the field names and type values exactly. Write the text in %s.`,
		f.Date, f.TodayWorkMin, f.TodayBreakMin, f.TodayPomodoros, string(week), i18n.ResponseLanguage())
}

// TemplateBriefing is the briefing without AI.
func TemplateBriefing(f Facts) string {
	parts := []string{fmt.Sprintf(i18n.Text("Yesterday: %s of work, %s of breaks."), reminder.FormatMinutes(f.YesterdayWorkMin), reminder.FormatMinutes(f.YesterdayBreakMin))}
	if f.NextEventInMin > 0 {
		parts = append(parts, fmt.Sprintf(i18n.Text("%d events today; the next one is in %d min."), f.TodayEvents, f.NextEventInMin))
	} else if f.TodayEvents > 0 {
		parts = append(parts, fmt.Sprintf(i18n.Text("%d events today."), f.TodayEvents))
	}
	if f.FocusPomodoros > 0 {
		parts = append(parts, fmt.Sprintf(i18n.Text("You have %d min to focus: %d pomodoros fit."), f.FocusMinutes, f.FocusPomodoros))
	}
	return strings.Join(parts, " ")
}

// TemplateRetro is the retrospective without AI.
func TemplateRetro(f Facts) string {
	s := fmt.Sprintf(i18n.Text("Today: %s of work, %s of breaks."), reminder.FormatMinutes(f.TodayWorkMin), reminder.FormatMinutes(f.TodayBreakMin))
	if f.TodayPomodoros > 0 {
		s += " " + fmt.Sprintf(i18n.Text("%d pomodoros done."), f.TodayPomodoros)
	}
	return s
}

// ParseRetro reads the retro JSON; plain text is taken as the retro alone.
func ParseRetro(raw string) (string, *insights.Report, error) {
	cleaned := strings.TrimSpace(raw)
	if !strings.Contains(cleaned, "{") {
		return cleaned, nil, nil
	}
	report, err := insights.ParseResponse(cleaned)
	if err != nil {
		return "", nil, err
	}
	var withRetro struct {
		Retro string `json:"retro"`
	}
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(cleaned, "```json"), "```"), "```"))
	_ = json.Unmarshal([]byte(trimmed), &withRetro)
	if strings.TrimSpace(withRetro.Retro) == "" {
		return "", nil, errors.New(i18n.Text("the retrospective is missing from the AI reply"))
	}
	return strings.TrimSpace(withRetro.Retro), report, nil
}

// Store keeps one JSON file per day.
type Store struct{ Dir string }

func (s Store) path(date string) string { return filepath.Join(s.Dir, date+".json") }

// Load returns the day, empty when nothing was written yet.
func (s Store) Load(date string) (Day, error) {
	d := Day{Date: date}
	data, err := os.ReadFile(s.path(date))
	if errors.Is(err, os.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(data, &d)
	return d, err
}

// Save writes the day, owner-only.
func (s Store) Save(d Day) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp := s.path(d.Date) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(d.Date))
}
