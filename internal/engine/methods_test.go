package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/reminder"
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
