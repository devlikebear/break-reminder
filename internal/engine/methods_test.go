package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/secrets"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

func call(t *testing.T, s *Server, out *lines, line string) map[string]any {
	t.Helper()
	out.Reset()
	if err := s.Serve(context.Background(), strings.NewReader(line+"\n")); err != nil {
		t.Fatal(err)
	}
	msgs := out.messages(t)
	for _, m := range msgs {
		if _, ok := m["id"]; ok {
			return m
		}
	}
	t.Fatalf("no response in %v", msgs)
	return nil
}

func TestRPCMethodsRoundTrip(t *testing.T) {
	cfg := testConfig()
	e, _, _, _ := newTestEngine(t, cfg)
	e.opts.LoadConfig = func() (config.Config, error) { return cfg, nil }
	e.opts.SaveConfig = func(c config.Config) error { cfg = c; return nil }
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")

	if r := call(t, s, &out, `{"jsonrpc":"2.0","id":1,"method":"engine.ping"}`); r["result"].(map[string]any)["version"] != "test" {
		t.Fatalf("ping = %v", r)
	}
	r := call(t, s, &out, `{"jsonrpc":"2.0","id":2,"method":"status.get"}`)
	if r["result"].(map[string]any)["work_duration_sec"].(float64) != 60 {
		t.Fatalf("status = %v", r)
	}
	r = call(t, s, &out, `{"jsonrpc":"2.0","id":3,"method":"pomodoro.set","params":{"enabled":true,"work_min":20}}`)
	if r["error"] != nil || cfg.TimerMode != config.TimerModePomodoro || cfg.PomodoroWorkMin != 20 {
		t.Fatalf("pomodoro.set = %v, cfg = %+v", r, cfg)
	}
	r = call(t, s, &out, `{"jsonrpc":"2.0","id":4,"method":"timer.snooze"}`)
	if r["error"].(map[string]any)["code"].(float64) != CodeRejected {
		t.Fatalf("snooze outside break = %v", r)
	}
	r = call(t, s, &out, `{"jsonrpc":"2.0","id":5,"method":"break_screen.choose","params":{"mode":"notify"}}`)
	if r["error"] != nil || cfg.BreakScreenMode != "notify" {
		t.Fatalf("break_screen.choose = %v", r)
	}
	r = call(t, s, &out, `{"jsonrpc":"2.0","id":6,"method":"timer.pause","params":{"reason":"focus","duration_sec":60}}`)
	if r["result"].(map[string]any)["state"].(map[string]any)["paused"] != true {
		t.Fatalf("pause = %v", r)
	}
}

func TestTimeToolCompletionBecomesEvent(t *testing.T) {
	e, c, r, _ := newTestEngine(t, testConfig())
	snap, err := e.TimeTool(TimeToolRequest{Kind: "start", DurationSec: 1, Label: "tea"})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Countdown == nil || snap.Countdown.Phase != "running" {
		t.Fatalf("start = %+v", snap.Countdown)
	}
	c.Advance(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go e.RunTimeTools(ctx)
	for ctx.Err() == nil {
		for _, typ := range r.types() {
			if typ == reminder.TypeTimeToolDone {
				r.mu.Lock()
				body := r.events[len(r.events)-1].Body
				r.mu.Unlock()
				if !strings.Contains(body, "tea") {
					t.Fatalf("body = %q", body)
				}
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no time_tool_done event, got %v", r.types())
}

func TestStatusJSONHasStateFields(t *testing.T) {
	e, _, _, _ := newTestEngine(t, testConfig())
	st, err := e.Status()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(st)
	for _, key := range []string{`"mode":"work"`, `"work_seconds"`, `"time_tools"`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("status JSON missing %s: %s", key, data)
		}
	}
}

func TestSecretsSetOnlyReachesTheOverrideLayer(t *testing.T) {
	e, _, _, _ := newTestEngine(t, testConfig())
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")
	t.Cleanup(func() { _ = secrets.Overrides.Delete(secrets.TTSAPIKey) })

	r := call(t, s, &out, `{"jsonrpc":"2.0","id":1,"method":"secrets.set","params":{"name":"tts_api_key","value":"AIza-from-app"}}`)
	if r["error"] != nil {
		t.Fatalf("secrets.set = %v", r)
	}
	if v, _ := secrets.Overrides.Get(secrets.TTSAPIKey); v != "AIza-from-app" {
		t.Fatalf("override = %q", v)
	}
	if strings.Contains(out.String(), "AIza-from-app") {
		t.Fatal("secret echoed on stdout")
	}
	r = call(t, s, &out, `{"jsonrpc":"2.0","id":2,"method":"secrets.set","params":{"name":"voice","value":"x"}}`)
	if r["error"] == nil {
		t.Fatal("unknown secret name accepted")
	}
}

func TestCalendarRPCDrivesMeetingPauseAndStatus(t *testing.T) {
	cfg := awareConfig()
	e, c, r, _ := newTestEngine(t, cfg)
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")

	now := c.Now().Unix()
	e.ReportIdle(0) // opens the session
	update := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"context.calendar.update","params":{"events":[
		{"id":"m1","title":"Standup","start":%d,"end":%d,"has_attendees":true,"availability":"busy"},
		{"id":"m2","title":"Review","start":%d,"end":%d,"has_attendees":true,"availability":"busy"}]}}`,
		now+60, now+1800, now+3600, now+5400)
	if resp := call(t, s, &out, strings.ReplaceAll(update, "\n", "")); resp["error"] != nil {
		t.Fatalf("update = %v", resp)
	}
	st, _ := e.Status()
	if st.Context.Calendar.Stale || st.Context.Calendar.Next == nil || st.Context.Calendar.Next.ID != "m1" || st.Context.Calendar.FocusMinutes != 1 {
		t.Fatalf("context before = %+v", st.Context.Calendar)
	}

	c.Advance(2 * time.Minute)
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	st, _ = e.Status()
	if !st.State.Paused || st.State.PauseSource != "calendar" || st.Context.Calendar.Current == nil {
		t.Fatalf("not paused for m1: state=%+v ctx=%+v", st.State, st.Context.Calendar)
	}
	// Manual resume holds for this meeting.
	if _, err := e.Resume(); err != nil {
		t.Fatal(err)
	}
	c.Advance(time.Minute)
	e.ReportIdle(0)
	_ = e.Tick()
	if st, _ = e.Status(); st.State.Paused {
		t.Fatal("re-paused after manual resume")
	}
	_ = r
}

func TestAppUsageRPC(t *testing.T) {
	e, _, _, dir := newTestEngine(t, testConfig())
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")
	resp := call(t, s, &out, `{"jsonrpc":"2.0","id":1,"method":"context.app_usage.add","params":{"items":[{"bundle_id":"com.apple.Safari","name":"Safari","seconds":60},{"bundle_id":"com.microsoft.VSCode","name":"Code","seconds":120}]}}`)
	if resp["error"] != nil {
		t.Fatalf("add = %v", resp)
	}
	st, _ := e.Status()
	if len(st.Context.AppUsageToday) != 2 || st.Context.AppUsageToday[0].Name != "Code" {
		t.Fatalf("usage = %+v", st.Context.AppUsageToday)
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, "app-usage")); len(entries) != 1 {
		t.Fatalf("files = %v", entries)
	}
	call(t, s, &out, `{"jsonrpc":"2.0","id":2,"method":"context.app_usage.clear"}`)
	if st, _ = e.Status(); len(st.Context.AppUsageToday) != 0 {
		t.Fatal("clear left data")
	}
}

func TestAppUsageIgnoredWhenDisabled(t *testing.T) {
	cfg := testConfig()
	cfg.AppUsageEnabled = false
	e, _, _, _ := newTestEngine(t, cfg)
	if err := e.AddAppUsage([]workcontext.AppUsage{{BundleID: "a", Name: "A", Seconds: 5}}); err != nil {
		t.Fatal(err)
	}
	if st, _ := e.Status(); len(st.Context.AppUsageToday) != 0 {
		t.Fatal("usage recorded while disabled")
	}
}

func TestContextSettingsRPC(t *testing.T) {
	cfg := testConfig()
	e, _, _, _ := newTestEngine(t, cfg)
	e.opts.LoadConfig = func() (config.Config, error) { return cfg, nil }
	e.opts.SaveConfig = func(c config.Config) error { cfg = c; return nil }
	var out lines
	s := NewServer(&out)
	e.Register(s, "test")
	resp := call(t, s, &out, `{"jsonrpc":"2.0","id":1,"method":"context.settings.set","params":{"calendar_aware_breaks":true,"meeting_rule":"busy","calendar_excluded_ids":["x"],"app_usage_enabled":false}}`)
	if resp["error"] != nil || !cfg.CalendarAwareBreaks || cfg.MeetingRule != "busy" || len(cfg.CalendarExcludedIDs) != 1 || cfg.AppUsageEnabled {
		t.Fatalf("resp=%v cfg=%+v", resp, cfg)
	}
	resp = call(t, s, &out, `{"jsonrpc":"2.0","id":2,"method":"context.settings.set","params":{"meeting_rule":"often"}}`)
	if resp["error"] == nil {
		t.Fatal("invalid rule accepted")
	}
}
