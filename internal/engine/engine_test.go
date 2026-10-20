package engine

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestMain(m *testing.M) {
	os.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	os.Exit(m.Run())
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

type recorder struct {
	mu       sync.Mutex
	events   []reminder.Event
	statuses int
}

func (r *recorder) emit(e reminder.Event) { r.mu.Lock(); r.events = append(r.events, e); r.mu.Unlock() }
func (r *recorder) status(Status)         { r.mu.Lock(); r.statuses++; r.mu.Unlock() }
func (r *recorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		out = append(out, e.Type)
	}
	return out
}

func testConfig() config.Config {
	cfg := config.Default()
	cfg.WorkDurationMin = 1
	cfg.BreakDurationMin = 1
	cfg.AutoSessionDetect = true
	cfg.SessionDetectStartHour = 0
	cfg.SessionDetectEndHour = 24
	cfg.WorkDays = []int{0, 1, 2, 3, 4, 5, 6}
	cfg.BreakScreenMode = "block"
	cfg.TTSEnabled = true
	return cfg
}

func newTestEngine(t *testing.T, cfg config.Config) (*Engine, *clock, *recorder, string) {
	t.Helper()
	dir := t.TempDir()
	c := &clock{t: time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)}
	r := &recorder{}
	e := New(Options{
		Now:          c.Now,
		LoadConfig:   func() (config.Config, error) { return cfg, nil },
		StatePath:    filepath.Join(dir, "state"),
		LogPath:      filepath.Join(dir, "log"),
		TimeToolsDir: dir,
		Emit:         r.emit,
		Status:       r.status,
	})
	return e, c, r, dir
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestTickRunsTheCycleWithReportedIdle(t *testing.T) {
	e, c, r, _ := newTestEngine(t, testConfig())

	e.ReportIdle(0)
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	if got := r.types(); !contains(got, reminder.TypeSessionStart) {
		t.Fatalf("first active tick should open a session, got %v", got)
	}

	c.Advance(60 * time.Second)
	e.ReportIdle(0)
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	got := r.types()
	if !contains(got, reminder.TypeBreakStart) || !contains(got, reminder.TypeSpeak) {
		t.Fatalf("break should start after 1 work minute, got %v", got)
	}
	st, err := e.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.State.Mode != "break" || st.WorkDurationSec != 60 || st.BreakDurationSec != 60 {
		t.Fatalf("status = %+v", st)
	}
	if r.statuses == 0 {
		t.Fatal("ticks should publish status")
	}

	// Sit idle through the break: it ends and work resumes.
	c.Advance(70 * time.Second)
	e.ReportIdle(70)
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	if !contains(r.types(), reminder.TypeBreakOver) {
		t.Fatalf("break should end, got %v", r.types())
	}
}

func TestIdleExtrapolatesFromLastReport(t *testing.T) {
	e, c, _, _ := newTestEngine(t, testConfig())
	if got := e.idleSeconds(c.Now()); got != 0 {
		t.Fatalf("idle without any report = %d, want 0", got)
	}
	e.ReportIdle(30)
	c.Advance(45 * time.Second)
	if got := e.idleSeconds(c.Now()); got != 75 {
		t.Fatalf("idle = %d, want 75", got)
	}
}

func TestControlMethodsMutateState(t *testing.T) {
	e, c, _, _ := newTestEngine(t, testConfig())

	if _, err := e.StartSession(); err != nil {
		t.Fatal(err)
	}
	st, _ := e.Status()
	if !st.State.IsSessionActive() || !st.State.SessionManual {
		t.Fatalf("session.start: %+v", st.State)
	}

	if _, err := e.Pause(state.PauseReasonMeeting, 600); err != nil {
		t.Fatal(err)
	}
	st, _ = e.Status()
	if !st.State.Paused || st.State.PauseUntil != c.Now().Unix()+600 {
		t.Fatalf("pause: %+v", st.State)
	}
	if _, err := e.Pause("nap", 0); err == nil {
		t.Fatal("invalid pause reason accepted")
	}
	if _, err := e.Resume(); err != nil {
		t.Fatal(err)
	}
	st, _ = e.Status()
	if st.State.Paused {
		t.Fatal("resume did not clear pause")
	}

	if _, err := e.Snooze(5 * time.Minute); err == nil {
		t.Fatal("snooze outside a break should be rejected")
	}

	if _, err := e.StopSession(); err != nil {
		t.Fatal(err)
	}
	st, _ = e.Status()
	if st.State.SessionState != state.SessionStateEnded {
		t.Fatalf("session.stop: %+v", st.State)
	}

	if _, err := e.Reset(); err != nil {
		t.Fatal(err)
	}
	st, _ = e.Status()
	if st.State.SessionState != "" || st.State.Mode != "work" {
		t.Fatalf("reset: %+v", st.State)
	}
}

func TestTickSavesDailyHistoryOnNewDay(t *testing.T) {
	e, c, _, _ := newTestEngine(t, testConfig())
	var saved []string
	e.opts.AppendHistory = func(date string, work, brk int, hourly [24]int) error {
		saved = append(saved, date)
		return nil
	}
	s := state.New()
	s.LastUpdateDate = "2026-10-19"
	s.TodayWorkSeconds = 3600
	s.LastCheck = c.Now().Unix() - 60
	if err := state.Save(e.opts.StatePath, s); err != nil {
		t.Fatal(err)
	}
	e.ReportIdle(0)
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0] != "2026-10-19" {
		t.Fatalf("saved = %v", saved)
	}
}

func TestTickWaitsForFirstIdleReport(t *testing.T) {
	e, _, r, _ := newTestEngine(t, testConfig())
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	if got := r.types(); len(got) != 0 {
		t.Fatalf("tick before any idle report emitted %v", got)
	}
	st, _ := e.Status()
	if st.State.IsSessionActive() {
		t.Fatal("a session must not start before the app reports idle time")
	}
	// The first report ticks right away.
	e.ReportIdle(0)
	if got := r.types(); !contains(got, reminder.TypeSessionStart) {
		t.Fatalf("first idle report should tick, got %v", got)
	}
}
