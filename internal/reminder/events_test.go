package reminder

import (
	"os"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timer"
)

func TestMain(m *testing.M) {
	os.Setenv("BREAK_REMINDER_LANGUAGE", "en") // assert the English text contract
	os.Exit(m.Run())
}

func types(events []Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}

func TestBreakTimeCarriesOverlayAndSpeech(t *testing.T) {
	cfg := config.Default()
	cfg.BreakScreenMode = "block"
	cfg.Voice = "Yuna"
	s := state.New()
	s.Mode = "break"
	s.BreakStart = 1000
	s.TodayWorkSeconds = 3000
	s.TodayBreakSeconds = 600
	got := Events(cfg, timer.TickResult{State: s, Actions: []timer.Action{timer.ActionNotifyBreakTime, timer.ActionSpeakBreakTime}})
	if strings.Join(types(got), ",") != "break_start,speak" {
		t.Fatalf("types = %v", types(got))
	}
	b := got[0]
	if b.ScreenMode != "block" || b.BreakStart != 1000 || b.BreakSec != cfg.EffectiveBreakSec(0) {
		t.Fatalf("break_start = %+v", b)
	}
	if b.TodayWorkMin != 50 || b.TodayBreakMin != 10 {
		t.Fatalf("today stats = %d/%d", b.TodayWorkMin, b.TodayBreakMin)
	}
	if b.Title != "Break Time!" || b.Body == "" {
		t.Fatalf("break_start fallback notification = %q / %q", b.Title, b.Body)
	}
	if got[1].Voice != "Yuna" || !strings.Contains(got[1].Text, "Time for a break") {
		t.Fatalf("speak = %+v", got[1])
	}
}

func TestNotificationsKeepLegacyText(t *testing.T) {
	cfg := config.Default()
	sum := &timer.SessionSummary{WorkSeconds: 3600, BreakSeconds: 600}
	got := Events(cfg, timer.TickResult{
		State:        state.New(),
		Actions:      []timer.Action{timer.ActionNotifyBreakOver, timer.ActionNotifyFiveMinWarning, timer.ActionNotifyStillOnBreak, timer.ActionNotifySessionStart, timer.ActionNotifySessionEnd},
		SessionEnded: sum,
	})
	want := []struct{ typ, title, sound string }{
		{"break_over", "Break Over!", "Hero"},
		{"five_min_warning", "5 minutes left", ""},
		{"still_on_break", "Still on break!", ""},
		{"session_start", "Work session started", "Submarine"},
		{"session_end", "Work session ended", "Glass"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", types(got))
	}
	for i, w := range want {
		if got[i].Type != w.typ || got[i].Title != w.title || got[i].Sound != w.sound {
			t.Errorf("event %d = %+v, want %+v", i, got[i], w)
		}
	}
	if got[4].Body != "Today · Work 1h · Break 10m" {
		t.Errorf("session_end body = %q", got[4].Body)
	}
}

func TestSilentActionsProduceNoEvents(t *testing.T) {
	got := Events(config.Default(), timer.TickResult{State: state.New(), Actions: []timer.Action{timer.ActionSaveDailyHistory, timer.ActionNone}})
	if len(got) != 0 {
		t.Fatalf("got %v", types(got))
	}
}

func TestSessionEndMessageWithoutSummary(t *testing.T) {
	if got := SessionEndMessage(nil); got != "Timer stopped. See you next session!" {
		t.Fatalf("got %q", got)
	}
}
