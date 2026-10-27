package engine

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/agent"
	"github.com/devlikebear/break-reminder/internal/assistant"
	"github.com/devlikebear/break-reminder/internal/insights"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

type fakeAI struct {
	mu      sync.Mutex
	prompts []string
	reply   string
	err     error
}

func (f *fakeAI) Write(_ context.Context, _, prompt string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, prompt)
	return f.reply, f.err
}
func (f *fakeAI) Ask(context.Context, string, []agent.Turn, bool, func(string)) (agent.Answer, error) {
	return agent.Answer{Text: "answer"}, nil
}
func (f *fakeAI) Status() agent.ProviderStatus {
	return agent.ProviderStatus{Kind: "claude-code-cli", Ready: true}
}
func (f *fakeAI) Test(context.Context) error  { return nil }
func (f *fakeAI) SetKey(string, string) error { return nil }
func (f *fakeAI) count() int                  { f.mu.Lock(); defer f.mu.Unlock(); return len(f.prompts) }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !cond() {
		t.Fatal("condition not met in time")
	}
}

func findEvent(r *recorder, typ string) *reminder.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.events {
		if r.events[i].Type == typ {
			e := r.events[i]
			return &e
		}
	}
	return nil
}

func TestBriefingOncePerDayOnSessionStart(t *testing.T) {
	cfg := testConfig()
	cfg.BriefingVoice = true
	e, _, r, dir := newTestEngine(t, cfg)
	ai := &fakeAI{reply: "Good morning. 3 events today."}
	e.opts.AI = ai

	e.ReportIdle(0) // auto-detected first activity opens the session
	waitFor(t, func() bool { return findEvent(r, reminder.TypeBriefing) != nil })
	ev := findEvent(r, reminder.TypeBriefing)
	if ev.Body != "Good morning. 3 events today." || ev.Text != ev.Body {
		t.Fatalf("briefing event = %+v", ev)
	}
	day, _ := assistant.Store{Dir: filepath.Join(dir, "assistant")}.Load(e.opts.Now().Format("2006-01-02"))
	if day.Briefing == "" || day.BriefingSource != assistant.SourceAI {
		t.Fatalf("stored day = %+v", day)
	}
	// A second session start the same day does not brief again.
	if _, err := e.StopSession(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.StartSession(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	briefings := 0
	for _, typ := range r.types() {
		if typ == reminder.TypeBriefing {
			briefings++
		}
	}
	if briefings != 1 {
		t.Fatalf("briefings = %d", briefings)
	}
	st, _ := e.Status()
	if st.Assistant.Briefing == "" {
		t.Fatal("status lacks the briefing card")
	}
	if _, err := e.DismissAssistant(); err != nil {
		t.Fatal(err)
	}
	if st, _ = e.Status(); !st.Assistant.Dismissed {
		t.Fatal("dismiss not kept")
	}
}

func TestBriefingFallsBackToTemplateAndStaysQuietInMeetings(t *testing.T) {
	cfg := testConfig()
	cfg.BriefingVoice = true
	e, c, r, _ := newTestEngine(t, cfg)
	e.opts.AI = &fakeAI{err: agent.ErrNoConsent}
	now := c.Now().Unix()
	_ = e.UpdateCalendar([]workcontext.Event{{ID: "m", Start: now - 60, End: now + 600, HasAttendees: true, Availability: "busy"}})
	if _, err := e.StartSession(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return findEvent(r, reminder.TypeBriefing) != nil })
	ev := findEvent(r, reminder.TypeBriefing)
	if ev.Text != "" {
		t.Fatalf("spoke during a meeting: %+v", ev)
	}
	if !strings.Contains(ev.Body, "Yesterday") {
		t.Fatalf("template briefing = %q", ev.Body)
	}
}

func TestRetroOnSessionEndWritesInsights(t *testing.T) {
	e, _, r, dir := newTestEngine(t, testConfig())
	saved := make(chan *insights.Report, 1)
	e.opts.SaveInsights = func(rep *insights.Report) error { saved <- rep; return nil }
	ai := &fakeAI{reply: `{"retro":"Solid day.","daily_report":"Steady week.","patterns":[]}`}
	e.opts.AI = ai
	if _, err := e.StartSession(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return findEvent(r, reminder.TypeBriefing) != nil })
	if _, err := e.StopSession(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return findEvent(r, reminder.TypeRetro) != nil })
	select {
	case rep := <-saved:
		if rep.DailyReport != "Steady week." {
			t.Fatalf("report = %+v", rep)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("insights not saved")
	}
	day, _ := assistant.Store{Dir: filepath.Join(dir, "assistant")}.Load(e.opts.Now().Format("2006-01-02"))
	if day.Retro != "Solid day." {
		t.Fatalf("retro = %q", day.Retro)
	}
	if ai.count() != 2 {
		t.Fatalf("AI calls = %d, want briefing + retro", ai.count())
	}
	// The finished day carries the retro into history.
	if got := e.retroFor(e.opts.Now().Format("2006-01-02")); got != "Solid day." {
		t.Fatalf("history retro = %q", got)
	}
}

func TestBackendViews(t *testing.T) {
	cfg := testConfig()
	e, c, _, _ := newTestEngine(t, cfg)
	e.ReportIdle(0)
	b := e.Backend()
	tv, err := b.Timer()
	if err != nil || tv.Phase != "working" || tv.WorkTargetMin != 1 || !tv.SessionActive {
		t.Fatalf("timer = %+v %v", tv, err)
	}
	now := c.Now().Unix()
	_ = e.UpdateCalendar([]workcontext.Event{{ID: "n", Title: "Review", Start: now + 1800, End: now + 3600, Availability: "busy"}})
	cv, _ := b.Calendar()
	if cv.Next == nil || cv.FocusMinutes != 30 || len(cv.Events) != 1 {
		t.Fatalf("calendar = %+v", cv)
	}
	_ = e.AddAppUsage([]workcontext.AppUsage{{BundleID: "a", Name: "A", Seconds: 60}})
	if apps, _ := b.AppUsage(3); len(apps) != 1 || apps[0].Seconds != 60 {
		t.Fatalf("usage = %+v", apps)
	}
	if err := b.Pause("focus", 600); err != nil {
		t.Fatal(err)
	}
	if tv, _ = b.Timer(); tv.Phase != "paused" || tv.PauseReason != "focus" {
		t.Fatalf("paused = %+v", tv)
	}
	if err := b.Resume(); err != nil {
		t.Fatal(err)
	}
	if err := b.StartTimer(300, "tea"); err != nil {
		t.Fatal(err)
	}
	if err := b.StartPomodoro(); err != nil {
		t.Fatal(err)
	}
	if st, _ := e.Status(); st.TimerMode != "pomodoro" || !st.State.IsSessionActive() {
		t.Fatalf("after start_pomodoro: mode=%s session=%v", st.TimerMode, st.State.SessionState)
	}
}

type askingAI struct {
	fakeAI
	gotQuestion string
	gotHistory  []agent.Turn
	gotActions  bool
	keys        map[string]string
}

func (a *askingAI) Ask(_ context.Context, q string, h []agent.Turn, allow bool, onDelta func(string)) (agent.Answer, error) {
	a.gotQuestion, a.gotHistory, a.gotActions = q, h, allow
	onDelta("Two ")
	onDelta("pomodoros.")
	return agent.Answer{Text: "Two pomodoros.", History: append(h, agent.Turn{Role: "user", Content: q}, agent.Turn{Role: "assistant", Content: "Two pomodoros."})}, nil
}
func (a *askingAI) SetKey(kind, key string) error { a.keys[kind] = key; return nil }

func TestAIRPCs(t *testing.T) {
	e, _, _, _ := newTestEngine(t, testConfig())
	ai := &askingAI{keys: map[string]string{}}
	e.opts.AI = ai
	var notes []string
	var mu sync.Mutex
	e.opts.Notify = func(method string, params any) {
		mu.Lock()
		defer mu.Unlock()
		if d, ok := params.(AgentDelta); ok && method == NotifyAgentDelta {
			notes = append(notes, d.RequestID+":"+d.Text)
		}
	}
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")

	resp := call(t, s, &out, `{"jsonrpc":"2.0","id":1,"method":"agent.ask","params":{"request_id":"r1","question":"what now?","history":[{"role":"user","content":"hi"},{"role":"assistant","content":"hello"}]}}`)
	res, _ := resp["result"].(map[string]any)
	if res["text"] != "Two pomodoros." || ai.gotQuestion != "what now?" || len(ai.gotHistory) != 2 || ai.gotActions {
		t.Fatalf("ask = %v (q=%q h=%v actions=%v)", resp, ai.gotQuestion, ai.gotHistory, ai.gotActions)
	}
	mu.Lock()
	if strings.Join(notes, "|") != "r1:Two |r1:pomodoros." {
		t.Fatalf("deltas = %v", notes)
	}
	mu.Unlock()

	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":2,"method":"ai.key.set","params":{"provider":"anthropic","key":"sk-x"}}`); r["error"] != nil || ai.keys["anthropic"] != "sk-x" {
		t.Fatalf("key.set = %v", r)
	}
	if strings.Contains(out.String(), "sk-x") {
		t.Fatal("key echoed")
	}
	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":3,"method":"ai.consent.set","params":{"provider":"anthropic","consent":true}}`); r["error"] != nil {
		t.Fatalf("consent = %v", r)
	}
	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":4,"method":"ai.settings.set","params":{"ai_provider":"anthropic","ai_enabled":true,"share_event_titles":true}}`); r["error"] != nil {
		t.Fatalf("settings = %v", r)
	}
	cfg, _ := e.opts.LoadConfig()
	if cfg.AIProvider != "anthropic" || !cfg.AIEnabled || !cfg.ShareEventTitles || len(cfg.AIConsentedProviders) != 1 {
		t.Fatalf("cfg = %+v", cfg)
	}
	call(t, s, &out, `{"jsonrpc":"2.0","id":5,"method":"ai.consent.set","params":{"provider":"anthropic","consent":false}}`)
	if cfg, _ = e.opts.LoadConfig(); len(cfg.AIConsentedProviders) != 0 {
		t.Fatalf("consent not withdrawn: %v", cfg.AIConsentedProviders)
	}
	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":6,"method":"ai.settings.set","params":{"voice":"x"}}`); r["error"] == nil {
		t.Fatal("unrelated key accepted")
	}
	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":7,"method":"suggestion.run","params":{"action":"start_timer","minutes":10,"label":"stretch"}}`); r["error"] != nil {
		t.Fatalf("suggestion.run = %v", r)
	}
	if st, _ := e.Status(); st.TimeTools == nil || st.TimeTools.Countdown == nil {
		t.Fatal("suggested timer did not start")
	}
	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":8,"method":"ai.status"}`); r["error"] != nil {
		t.Fatalf("ai.status = %v", r)
	}
}

func TestRetroWaitsForTheEndOfTheDay(t *testing.T) {
	cfg := testConfig() // work_end_hour 18
	at := func(h int) time.Time { return time.Date(2026, 10, 27, h, 0, 0, 0, time.Local) }
	cases := []struct {
		reason string
		now    time.Time
		want   bool
	}{
		{"manual", at(12), true},
		{"window", at(22), true},
		{"idle", at(12), false}, // lunch
		{"idle", at(19), true},
	}
	for _, c := range cases {
		if got := retroDue(cfg, c.now, c.reason); got != c.want {
			t.Errorf("retroDue(%s at %d:00) = %v, want %v", c.reason, c.now.Hour(), got, c.want)
		}
	}
}
