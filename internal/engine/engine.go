// Package engine is the long-running timer core used by the menu bar app.
//
// It reuses timer.Tick and the state file exactly like the launchd `check`
// job, but it never touches the OS: idle time is reported by the app and
// reminders leave as events for the app to show. That keeps the same engine
// usable inside the App Store sandbox.
package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timer"
	"github.com/devlikebear/break-reminder/internal/timetools"
)

// Options wires the engine to its clock, files and outputs.
type Options struct {
	Now          func() time.Time
	LoadConfig   func() (config.Config, error)
	SaveConfig   func(config.Config) error
	StatePath    string
	LogPath      string
	TimeToolsDir string
	// Emit receives reminder events (notifications, speech, break screen).
	Emit func(reminder.Event)
	// Status receives a fresh snapshot after every tick and mutation.
	Status func(Status)
	// AppendHistory stores a finished day; defaults to the AI history file.
	AppendHistory func(date string, workSec, breakSec int, hourlySec [24]int) error
}

// Engine owns the timer state while it runs.
type Engine struct {
	opts Options

	mu        sync.Mutex
	idleSec   int
	idleAt    time.Time
	idleKnown bool
	cfgErr    error
}

// Status is what the app needs to draw the popover.
type Status struct {
	Now                    int64               `json:"now"`
	State                  state.State         `json:"state"`
	Active                 bool                `json:"active"`
	TimerMode              string              `json:"timer_mode"`
	WorkDurationSec        int                 `json:"work_duration_sec"`
	BreakDurationSec       int                 `json:"break_duration_sec"`
	PomodoroWorkMin        int                 `json:"pomodoro_work_min"`
	PomodoroBreakMin       int                 `json:"pomodoro_break_min"`
	PomodoroLongBreakMin   int                 `json:"pomodoro_long_break_min"`
	PomodoroLongBreakEvery int                 `json:"pomodoro_long_break_every"`
	IdleSec                int                 `json:"idle_sec"`
	IdleThresholdSec       int                 `json:"idle_threshold_sec"`
	BreakScreenMode        string              `json:"break_screen_mode"`
	NotificationsEnabled   bool                `json:"notifications_enabled"`
	AutoSessionDetect      bool                `json:"auto_session_detect"`
	CheckIntervalSec       int                 `json:"check_interval_sec"`
	TimeTools              *timetools.Snapshot `json:"time_tools,omitempty"`
	ConfigError            string              `json:"config_error,omitempty"`
}

// New builds an engine; zero-valued options fall back to real defaults.
func New(opts Options) *Engine {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.LoadConfig == nil {
		opts.LoadConfig = config.Load
	}
	if opts.SaveConfig == nil {
		opts.SaveConfig = config.Save
	}
	if opts.StatePath == "" {
		opts.StatePath = state.DefaultStatePath()
	}
	if opts.LogPath == "" {
		opts.LogPath = logging.DefaultLogPath()
	}
	if opts.TimeToolsDir == "" {
		opts.TimeToolsDir = timetools.DefaultDirectory()
	}
	if opts.Emit == nil {
		opts.Emit = func(reminder.Event) {}
	}
	if opts.Status == nil {
		opts.Status = func(Status) {}
	}
	if opts.AppendHistory == nil {
		opts.AppendHistory = appendAIHistory
	}
	return &Engine{opts: opts}
}

func appendAIHistory(date string, workSec, breakSec int, hourlySec [24]int) error {
	var hourlyMin [24]int
	for i, s := range hourlySec {
		hourlyMin[i] = s / 60
	}
	return ai.AppendHistory(ai.DailySummary{Date: date, WorkMin: workSec / 60, BreakMin: breakSec / 60, HourlyWork: hourlyMin})
}

// config loads the current settings, keeping defaults when the file is broken
// so the timer never stops because of a typo.
func (e *Engine) config() config.Config {
	cfg, err := e.opts.LoadConfig()
	e.cfgErr = err
	if err != nil {
		return config.Default()
	}
	return cfg
}

// ReportIdle records how long the user has been idle, as measured by the app.
// The first report ticks at once, since ticks wait for it.
func (e *Engine) ReportIdle(sec int) {
	if sec < 0 {
		sec = 0
	}
	e.mu.Lock()
	first := !e.idleKnown
	e.idleSec, e.idleAt, e.idleKnown = sec, e.opts.Now(), true
	e.mu.Unlock()
	if first {
		if err := e.Tick(); err != nil {
			logging.Log(e.opts.LogPath, i18n.Text("Check failed")+": "+err.Error())
		}
	}
}

// idleSeconds extrapolates the last report: if the app stops reporting, the
// user is treated as idle for that time rather than as working.
func (e *Engine) idleSeconds(now time.Time) int {
	if !e.idleKnown {
		return 0
	}
	since := int(now.Sub(e.idleAt) / time.Second)
	if since < 0 {
		since = 0
	}
	return e.idleSec + since
}

// Tick advances the timer once, like one launchd `check` run. Until the app
// has reported idle time once, nothing advances: guessing "active" would open
// sessions for a user who is away.
func (e *Engine) Tick() error {
	e.mu.Lock()
	if !e.idleKnown {
		e.mu.Unlock()
		e.publish()
		return nil
	}
	cfg := e.config()
	now := e.opts.Now()
	path := e.opts.StatePath

	if current, err := state.Load(path); err == nil && !timer.NeedsTick(cfg, current, now) {
		err := state.Update(path, func(s state.State) (state.State, error) {
			if !s.Paused {
				s.LastCheck = now.Unix()
			}
			return s, nil
		})
		e.mu.Unlock()
		e.publish()
		return err
	}

	idleSec := e.idleSeconds(now)
	var result timer.TickResult
	if err := state.Update(path, func(s state.State) (state.State, error) {
		result = timer.Tick(cfg, s, now, idleSec)
		return result.State, nil
	}); err != nil {
		logging.Log(e.opts.LogPath, i18n.Text("Failed to update state, using fresh state")+": "+err.Error())
		result = timer.Tick(cfg, state.New(), now, idleSec)
		if saveErr := state.Save(path, result.State); saveErr != nil {
			e.mu.Unlock()
			return saveErr
		}
	}
	if result.LogMsg != "" {
		logging.Log(e.opts.LogPath, result.LogMsg)
	}
	if d := result.DayEndSummary; d != nil {
		if err := e.opts.AppendHistory(d.Date, d.WorkSeconds, d.BreakSeconds, d.HourlyWork); err != nil {
			logging.Log(e.opts.LogPath, err.Error())
		}
	}
	logging.Rotate(e.opts.LogPath, cfg.MaxLogLines)
	events := reminder.Events(cfg, result)
	e.mu.Unlock()

	for _, ev := range events {
		e.opts.Emit(ev)
	}
	e.publish()
	return nil
}

// Status returns the current snapshot.
func (e *Engine) Status() (Status, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.statusLocked()
}

func (e *Engine) statusLocked() (Status, error) {
	cfg := e.config()
	now := e.opts.Now()
	s, err := state.Load(e.opts.StatePath)
	if err != nil {
		return Status{}, err
	}
	st := Status{
		Now:                    now.Unix(),
		State:                  s,
		Active:                 schedule.IsActive(cfg, s, now),
		TimerMode:              cfg.TimerMode,
		WorkDurationSec:        cfg.EffectiveWorkSec(),
		BreakDurationSec:       cfg.EffectiveBreakSec(s.PomodoroCount),
		PomodoroWorkMin:        cfg.PomodoroWorkMin,
		PomodoroBreakMin:       cfg.PomodoroBreakMin,
		PomodoroLongBreakMin:   cfg.PomodoroLongBreakMin,
		PomodoroLongBreakEvery: cfg.PomodoroLongBreakEvery,
		IdleSec:                e.idleSeconds(now),
		IdleThresholdSec:       cfg.IdleThresholdSec,
		BreakScreenMode:        cfg.BreakScreenMode,
		NotificationsEnabled:   cfg.NotificationsEnabled,
		AutoSessionDetect:      cfg.AutoSessionDetect,
		CheckIntervalSec:       cfg.CheckIntervalSec,
	}
	if snap, err := timetools.NewStore(e.opts.TimeToolsDir).Load(); err == nil {
		st.TimeTools = &snap
	}
	if e.cfgErr != nil {
		st.ConfigError = e.cfgErr.Error()
	}
	return st, nil
}

func (e *Engine) publish() {
	if st, err := e.Status(); err == nil {
		e.opts.Status(st)
	}
}

// mutate applies fn to the state under the engine lock, logs msg, and
// publishes the new status.
func (e *Engine) mutate(msg string, fn func(state.State, time.Time) (state.State, error)) (Status, error) {
	e.mu.Lock()
	now := e.opts.Now()
	err := state.Update(e.opts.StatePath, func(s state.State) (state.State, error) { return fn(s, now) })
	if err == nil && msg != "" {
		logging.Log(e.opts.LogPath, msg)
	}
	var st Status
	if err == nil {
		st, err = e.statusLocked()
	}
	e.mu.Unlock()
	if err != nil {
		return Status{}, err
	}
	e.opts.Status(st)
	return st, nil
}

// StartSession opens a manual work session (no-op when one is running).
func (e *Engine) StartSession() (Status, error) {
	return e.mutate(i18n.Text("Work session started (manual)"), func(s state.State, now time.Time) (state.State, error) {
		if s.IsSessionActive() {
			return s, nil
		}
		return s.StartSession(now.Unix(), true), nil
	})
}

// StopSession ends the session and holds reminders until the next start.
func (e *Engine) StopSession() (Status, error) {
	var summary *timer.SessionSummary
	st, err := e.mutate(i18n.Text("Work session stopped (manual)"), func(s state.State, now time.Time) (state.State, error) {
		if s.IsSessionActive() {
			sum := timer.EndSessionSummary(s, now.Unix(), timer.SessionEndManual)
			summary = &sum
		}
		return s.EndSession(now.Unix(), true), nil
	})
	if err == nil && summary != nil {
		e.opts.Emit(reminder.Event{Type: reminder.TypeSessionEnd, Title: i18n.Text("Work session ended"), Body: reminder.SessionEndMessage(summary), Sound: "Glass"})
	}
	return st, err
}

// Pause holds the timer; durationSec > 0 auto-resumes.
func (e *Engine) Pause(reason string, durationSec int) (Status, error) {
	if reason == "" {
		reason = state.PauseReasonMeeting
	}
	if !state.IsValidPauseReason(reason) {
		return Status{}, InvalidParams(fmt.Errorf(i18n.Text("invalid --mode %q (must be meeting|focus|afk)"), reason))
	}
	if durationSec < 0 {
		return Status{}, InvalidParams(errors.New(i18n.Text("snooze duration must be greater than zero")))
	}
	return e.mutate(fmt.Sprintf(i18n.Text("Timer paused (reason=%s, duration=%ds)"), reason, durationSec), func(s state.State, now time.Time) (state.State, error) {
		if s.Paused {
			return s, nil
		}
		return s.Pause(now.Unix(), reason, durationSec), nil
	})
}

// Resume continues a paused timer.
func (e *Engine) Resume() (Status, error) {
	return e.mutate(i18n.Text("Timer resumed"), func(s state.State, now time.Time) (state.State, error) {
		if !s.Paused {
			return s, nil
		}
		return s.Resume(now.Unix()), nil
	})
}

// Reset clears the timer state.
func (e *Engine) Reset() (Status, error) {
	return e.mutate(i18n.Text("Timer manually reset"), func(_ state.State, now time.Time) (state.State, error) {
		s := state.New()
		s.LastCheck = now.Unix()
		s.LastUpdateDate = now.Format("2006-01-02")
		return s, nil
	})
}

// Snooze ends the current break early and postpones the next one.
func (e *Engine) Snooze(d time.Duration) (Status, error) {
	st, err := e.mutate("", func(s state.State, now time.Time) (state.State, error) {
		return s.SnoozeBreak(now, d)
	})
	if err != nil {
		switch {
		case errors.Is(err, state.ErrBreakNotActive):
			return Status{}, Rejected(errors.New(i18n.Text("cannot snooze: no active break")))
		case errors.Is(err, state.ErrStatePaused), errors.Is(err, state.ErrInvalidSnooze):
			return Status{}, Rejected(err)
		}
	}
	return st, err
}

// Run ticks every interval until ctx ends. The first tick happens at once.
func (e *Engine) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := e.Tick(); err != nil {
			logging.Log(e.opts.LogPath, i18n.Text("Check failed")+": "+err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
