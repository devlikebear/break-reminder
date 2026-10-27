package mcphost

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

type fakeBackend struct {
	shareTitles bool
	calls       []string
	failPause   bool
}

func (f *fakeBackend) Timer() (TimerView, error) {
	return TimerView{Phase: "working", SessionActive: true, WorkMin: 20, WorkTargetMin: 50, TodayWorkMin: 140, TimerMode: "classic"}, nil
}
func (f *fakeBackend) Calendar() (CalendarView, error) {
	return CalendarView{
		Events:       []workcontext.Event{{ID: "a", Title: "1:1 with Mina", Start: 100, End: 200, HasAttendees: true, Availability: "busy"}},
		Next:         &workcontext.Event{ID: "a", Title: "1:1 with Mina", Start: 100, End: 200},
		FocusMinutes: 40, FocusPomodoros: 1,
	}, nil
}
func (f *fakeBackend) AppUsage(days int) ([]workcontext.AppUsage, error) {
	return []workcontext.AppUsage{{BundleID: "com.microsoft.VSCode", Name: "Code", Seconds: 3600 * days}}, nil
}
func (f *fakeBackend) WorkHistory(days int) ([]ai.DailySummary, error) {
	out := make([]ai.DailySummary, days)
	for i := range out {
		out[i] = ai.DailySummary{Date: "2026-10-2" + string(rune('0'+i)), WorkMin: 300}
	}
	return out, nil
}
func (f *fakeBackend) ShareEventTitles() bool { return f.shareTitles }
func (f *fakeBackend) StartPomodoro() error   { f.calls = append(f.calls, "pomodoro"); return nil }
func (f *fakeBackend) StartTimer(sec int, label string) error {
	f.calls = append(f.calls, "timer:"+label)
	return nil
}
func (f *fakeBackend) Pause(reason string, sec int) error {
	if f.failPause {
		return errors.New("already paused")
	}
	f.calls = append(f.calls, "pause:"+reason)
	return nil
}
func (f *fakeBackend) Resume() error { f.calls = append(f.calls, "resume"); return nil }

func connect(t *testing.T, srv *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func toolNames(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

func callText(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func TestReadOnlyServerExposesOnlyReadTools(t *testing.T) {
	h := &Host{Backend: &fakeBackend{}}
	got := toolNames(t, connect(t, h.Server(ServerOptions{})))
	want := append([]string(nil), ReadTools...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tools = %v, want %v", got, want)
	}
}

func TestActionServerAddsActionsAndSuggestions(t *testing.T) {
	var suggested []Suggestion
	h := &Host{Backend: &fakeBackend{}}
	got := toolNames(t, connect(t, h.Server(ServerOptions{Actions: true, Suggest: func(s Suggestion) { suggested = append(suggested, s) }})))
	for _, name := range append(append([]string{}, ActionTools...), SuggestTool) {
		found := false
		for _, g := range got {
			found = found || g == name
		}
		if !found {
			t.Fatalf("missing %s in %v", name, got)
		}
	}
}

func TestReadToolsReturnData(t *testing.T) {
	b := &fakeBackend{}
	cs := connect(t, (&Host{Backend: b}).Server(ServerOptions{}))

	text, isErr := callText(t, cs, "get_status", nil)
	if isErr || !strings.Contains(text, `"phase":"working"`) || !strings.Contains(text, `"today_work_min":140`) {
		t.Fatalf("get_status = %s", text)
	}
	text, _ = callText(t, cs, "get_focus_window", nil)
	if !strings.Contains(text, `"focus_minutes":40`) {
		t.Fatalf("get_focus_window = %s", text)
	}
	text, _ = callText(t, cs, "get_app_usage", map[string]any{"days": 3})
	if !strings.Contains(text, `"seconds":10800`) {
		t.Fatalf("get_app_usage = %s", text)
	}
	text, _ = callText(t, cs, "get_work_history", map[string]any{"days": 2})
	var hist struct {
		Days []ai.DailySummary `json:"days"`
	}
	if err := json.Unmarshal([]byte(text), &hist); err != nil || len(hist.Days) != 2 {
		t.Fatalf("get_work_history = %s (%v)", text, err)
	}
	if _, isErr := callText(t, cs, "get_app_usage", map[string]any{"days": 99}); !isErr {
		t.Fatal("days out of range accepted")
	}
}

func TestCalendarTitlesNeedConsentSetting(t *testing.T) {
	b := &fakeBackend{}
	cs := connect(t, (&Host{Backend: b}).Server(ServerOptions{}))
	text, _ := callText(t, cs, "get_today_calendar", nil)
	if strings.Contains(text, "Mina") || !strings.Contains(text, BusyTitle) {
		t.Fatalf("titles leaked without share_event_titles: %s", text)
	}
	b.shareTitles = true
	text, _ = callText(t, cs, "get_today_calendar", nil)
	if !strings.Contains(text, "1:1 with Mina") {
		t.Fatalf("titles hidden with share_event_titles: %s", text)
	}
}

func TestActionToolsCallBackendAndReportErrors(t *testing.T) {
	b := &fakeBackend{}
	cs := connect(t, (&Host{Backend: b}).Server(ServerOptions{Actions: true}))
	callText(t, cs, "start_pomodoro", nil)
	callText(t, cs, "start_timer", map[string]any{"minutes": 10, "label": "tea"})
	callText(t, cs, "pause", map[string]any{"mode": "focus", "minutes": 30})
	callText(t, cs, "resume", nil)
	if strings.Join(b.calls, ",") != "pomodoro,timer:tea,pause:focus,resume" {
		t.Fatalf("calls = %v", b.calls)
	}
	b.failPause = true
	if text, isErr := callText(t, cs, "pause", map[string]any{"mode": "meeting"}); !isErr || !strings.Contains(text, "already paused") {
		t.Fatalf("pause error = %q %v", text, isErr)
	}
	if _, isErr := callText(t, cs, "start_timer", map[string]any{"minutes": 0}); !isErr {
		t.Fatal("zero-minute timer accepted")
	}
}

func TestSuggestRecordsWithoutRunning(t *testing.T) {
	b := &fakeBackend{}
	var got []Suggestion
	cs := connect(t, (&Host{Backend: b}).Server(ServerOptions{Suggest: func(s Suggestion) { got = append(got, s) }}))
	text, isErr := callText(t, cs, SuggestTool, map[string]any{"action": "start_timer", "minutes": 15, "label": "stretch", "reason": "you have a gap"})
	if isErr || len(got) != 1 || got[0].Action != "start_timer" || got[0].Minutes != 15 || len(b.calls) != 0 {
		t.Fatalf("suggest = %q %v %+v calls=%v", text, isErr, got, b.calls)
	}
	if _, isErr := callText(t, cs, SuggestTool, map[string]any{"action": "delete_everything"}); !isErr {
		t.Fatal("unknown action accepted")
	}
}
