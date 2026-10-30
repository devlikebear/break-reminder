package engine

import (
	"sort"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// Backend exposes the engine to the MCP tool layer.
func (e *Engine) Backend() mcphost.Backend { return engineBackend{e} }

type engineBackend struct{ e *Engine }

func (b engineBackend) Timer() (mcphost.TimerView, error) {
	st, err := b.e.Status()
	if err != nil {
		return mcphost.TimerView{}, err
	}
	s := st.State
	v := mcphost.TimerView{
		SessionActive:  st.Active,
		TimerMode:      st.TimerMode,
		WorkMin:        s.WorkSeconds / 60,
		WorkTargetMin:  st.WorkDurationSec / 60,
		TodayWorkMin:   s.TodayWorkSeconds / 60,
		TodayBreakMin:  s.TodayBreakSeconds / 60,
		TodayPomodoros: s.TodayPomodoros,
	}
	switch {
	case s.Paused:
		v.Phase, v.PauseReason = "paused", s.PauseReason
	case !st.Active:
		v.Phase = "idle"
	case s.Mode == "break":
		v.Phase = "on_break"
		if left := st.BreakDurationSec - int(st.Now-s.BreakStart); left > 0 {
			v.BreakLeftMin = left / 60
		}
	default:
		v.Phase = "working"
	}
	return v, nil
}

func (b engineBackend) Calendar() (mcphost.CalendarView, error) {
	e := b.e
	e.mu.Lock()
	defer e.mu.Unlock()
	cfg := e.config()
	now := e.opts.Now()
	events := e.calendar.Visible(cfg.CalendarExcludedIDs)
	workcontext.SortEvents(events)
	v := mcphost.CalendarView{Events: events, Stale: e.calendar.Stale(now)}
	if !v.Stale {
		v.Current = workcontext.CurrentMeeting(events, now, cfg.MeetingRule)
		v.Next = workcontext.NextEvent(events, now)
		v.FocusMinutes, v.FocusPomodoros = workcontext.FocusWindow(events, now, cfg)
	}
	return v, nil
}

func (b engineBackend) AppUsage(days int) ([]workcontext.AppUsage, error) {
	e := b.e
	e.mu.Lock()
	cfg := e.config()
	now := e.opts.Now()
	e.mu.Unlock()
	if !cfg.AppUsageEnabled {
		return nil, nil
	}
	totals := map[string]workcontext.AppUsage{}
	for i := 0; i < days; i++ {
		items, err := e.usage.Day(now.AddDate(0, 0, -i))
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			t := totals[it.BundleID]
			t.BundleID, t.Seconds = it.BundleID, t.Seconds+it.Seconds
			if t.Name == "" {
				t.Name = it.Name
			}
			totals[it.BundleID] = t
		}
	}
	out := make([]workcontext.AppUsage, 0, len(totals))
	for _, t := range totals {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seconds > out[j].Seconds })
	if len(out) > 15 {
		out = out[:15]
	}
	return out, nil
}

func (b engineBackend) WorkHistory(days int) ([]ai.DailySummary, error) {
	hist, err := ai.LoadHistory()
	if err != nil {
		return nil, err
	}
	if len(hist) > days {
		hist = hist[len(hist)-days:]
	}
	return hist, nil
}

func (b engineBackend) ShareEventTitles() bool {
	b.e.mu.Lock()
	defer b.e.mu.Unlock()
	return b.e.config().ShareEventTitles
}

func (b engineBackend) StartPomodoro() error {
	if _, err := b.e.SetPomodoro(PomodoroRequest{Enabled: true}); err != nil {
		return err
	}
	_, err := b.e.StartSession()
	return err
}

func (b engineBackend) StartTimer(seconds int, label string) error {
	_, err := b.e.TimeTool(TimeToolRequest{Kind: "start", DurationSec: seconds, Label: label})
	return err
}

func (b engineBackend) Pause(reason string, seconds int) error {
	_, err := b.e.Pause(reason, seconds)
	return err
}

func (b engineBackend) Resume() error {
	_, err := b.e.Resume()
	return err
}
