package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/secrets"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/timetools"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// Notification method names sent from the engine to the app.
const (
	NotifyEvent  = "event"
	NotifyStatus = "status"
)

func decode(params json.RawMessage, v any) error {
	if err := json.Unmarshal(params, v); err != nil {
		return InvalidParams(err)
	}
	return nil
}

// Register installs every RPC method on s.
func (e *Engine) Register(s *Server, version string) {
	s.Handle("engine.ping", func(context.Context, json.RawMessage) (any, error) {
		return map[string]any{"version": version, "pid": os.Getpid()}, nil
	})
	s.Handle("status.get", func(context.Context, json.RawMessage) (any, error) { return e.Status() })
	s.Handle("idle.report", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Seconds int `json:"seconds"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		e.ReportIdle(req.Seconds)
		return nil, nil
	})
	s.Handle("session.start", func(context.Context, json.RawMessage) (any, error) { return e.StartSession() })
	s.Handle("session.stop", func(context.Context, json.RawMessage) (any, error) { return e.StopSession() })
	s.Handle("timer.pause", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Reason      string `json:"reason"`
			DurationSec int    `json:"duration_sec"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return e.Pause(req.Reason, req.DurationSec)
	})
	s.Handle("timer.resume", func(context.Context, json.RawMessage) (any, error) { return e.Resume() })
	s.Handle("timer.reset", func(context.Context, json.RawMessage) (any, error) { return e.Reset() })
	s.Handle("timer.snooze", func(_ context.Context, p json.RawMessage) (any, error) {
		req := struct {
			Seconds int `json:"seconds"`
		}{Seconds: 5 * 60}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return e.Snooze(time.Duration(req.Seconds) * time.Second)
	})
	s.Handle("pomodoro.set", func(_ context.Context, p json.RawMessage) (any, error) {
		var req PomodoroRequest
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return e.SetPomodoro(req)
	})
	s.Handle("break_screen.choose", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Mode string `json:"mode"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return e.ChooseBreakScreen(req.Mode)
	})
	s.Handle("secrets.set", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		if !config.IsSecretKey(req.Name) {
			return nil, InvalidParams(fmt.Errorf(i18n.Text("unknown config key %q"), req.Name))
		}
		// The sandboxed app keeps secrets in its own keychain; the engine only
		// holds them in memory.
		if req.Value == "" {
			return nil, secrets.Overrides.Delete(req.Name)
		}
		return nil, secrets.Overrides.Set(req.Name, req.Value)
	})
	s.Handle("context.calendar.update", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Events []workcontext.Event `json:"events"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return nil, e.UpdateCalendar(req.Events)
	})
	s.Handle("context.app_usage.add", func(_ context.Context, p json.RawMessage) (any, error) {
		var req struct {
			Items []workcontext.AppUsage `json:"items"`
		}
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		if err := e.AddAppUsage(req.Items); err != nil {
			return nil, InvalidParams(err)
		}
		return nil, nil
	})
	s.Handle("context.app_usage.clear", func(context.Context, json.RawMessage) (any, error) { return nil, e.ClearAppUsage() })
	s.Handle("context.settings.set", func(_ context.Context, p json.RawMessage) (any, error) {
		var req map[string]json.RawMessage
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		changes := map[string]any{}
		for key, raw := range req {
			switch key {
			case "calendar_aware_breaks", "app_usage_enabled", "meeting_rule", "calendar_excluded_ids":
				var v any
				if err := json.Unmarshal(raw, &v); err != nil {
					return nil, InvalidParams(err)
				}
				changes[key] = v
			default:
				return nil, InvalidParams(fmt.Errorf(i18n.Text("unknown config key %q"), key))
			}
		}
		if len(changes) == 0 {
			return e.Status()
		}
		if err := e.applyConfig(changes); err != nil {
			return nil, err
		}
		st, err := e.Status()
		if err == nil {
			e.opts.Status(st)
		}
		return st, err
	})
	s.Handle("time_tools.status", func(context.Context, json.RawMessage) (any, error) {
		return timetools.NewStore(e.opts.TimeToolsDir).Load()
	})
	s.Handle("time_tools.command", func(_ context.Context, p json.RawMessage) (any, error) {
		var req TimeToolRequest
		if err := decode(p, &req); err != nil {
			return nil, err
		}
		return e.TimeTool(req)
	})
}

// PomodoroRequest switches the timer mode; nil fields keep their value.
type PomodoroRequest struct {
	Enabled        bool `json:"enabled"`
	WorkMin        *int `json:"work_min,omitempty"`
	BreakMin       *int `json:"break_min,omitempty"`
	LongBreakMin   *int `json:"long_break_min,omitempty"`
	LongBreakEvery *int `json:"long_break_every,omitempty"`
}

// SetPomodoro mirrors `break-reminder pomodoro on|off`.
func (e *Engine) SetPomodoro(req PomodoroRequest) (Status, error) {
	changes := map[string]any{"timer_mode": config.TimerModeClassic}
	if req.Enabled {
		changes["timer_mode"] = config.TimerModePomodoro
		for key, v := range map[string]*int{
			"pomodoro_work_min":         req.WorkMin,
			"pomodoro_break_min":        req.BreakMin,
			"pomodoro_long_break_min":   req.LongBreakMin,
			"pomodoro_long_break_every": req.LongBreakEvery,
		} {
			if v != nil {
				changes[key] = *v
			}
		}
	}
	if err := e.applyConfig(changes); err != nil {
		return Status{}, err
	}
	// A mode switch restarts the running work block, like the CLI.
	return e.mutate("", func(s state.State, now time.Time) (state.State, error) {
		if s.Mode == "work" {
			s.WorkSeconds = 0
			s.SnoozeUntil = 0
			s.PomodoroCount = 0
			s.LastCheck = now.Unix()
		}
		return s, nil
	})
}

// ChooseBreakScreen stores the answer to the "ask" break screen prompt.
func (e *Engine) ChooseBreakScreen(mode string) (Status, error) {
	if mode != "block" && mode != "notify" {
		return Status{}, InvalidParams(fmt.Errorf(i18n.Text("Break screen mode must be block or notify, got %q"), mode))
	}
	if err := e.applyConfig(map[string]any{"break_screen_mode": mode}); err != nil {
		return Status{}, err
	}
	return e.Status()
}

func (e *Engine) applyConfig(changes map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	base, err := e.opts.LoadConfig()
	if err != nil {
		return Rejected(fmt.Errorf(i18n.Text("invalid config change: %w"), err))
	}
	data, err := yaml.Marshal(changes)
	if err != nil {
		return err
	}
	updated, err := config.ApplyYAMLChanges(base, data)
	if err != nil {
		return InvalidParams(fmt.Errorf(i18n.Text("invalid config change: %w"), err))
	}
	if err := e.opts.SaveConfig(updated); err != nil {
		return fmt.Errorf(i18n.Text("save config: %w"), err)
	}
	return nil
}

// TimeToolRequest mirrors the `time-tools timer ...` CLI flags.
type TimeToolRequest struct {
	Kind        string  `json:"kind"` // start pause resume cancel restart acknowledge notify-again
	ID          string  `json:"id,omitempty"`
	Label       string  `json:"label,omitempty"`
	DurationSec int     `json:"duration_sec,omitempty"`
	ReplaceID   string  `json:"replace_id,omitempty"`
	IfRevision  *uint64 `json:"if_revision,omitempty"`
}

// TimeTool applies one countdown command. The engine runs its own worker, so
// unlike the CLI there is no "worker stopped" precondition.
func (e *Engine) TimeTool(req TimeToolRequest) (timetools.Snapshot, error) {
	cmd := timetools.Command{Kind: req.Kind, ID: req.ID, Label: req.Label, ReplaceID: req.ReplaceID}
	switch req.Kind {
	case "start":
		if req.DurationSec < 1 || req.DurationSec > 24*60*60 {
			return timetools.Snapshot{}, InvalidParams(errors.New(i18n.Text("Enter a duration from 1 second to 24 hours in whole seconds")))
		}
		cmd.DurationMS = int64(req.DurationSec) * 1000
		id, err := timetools.NewID()
		if err != nil {
			return timetools.Snapshot{}, err
		}
		cmd.ID = id
	case "pause", "resume", "cancel", "restart", "acknowledge", "notify-again":
		if req.ID == "" {
			return timetools.Snapshot{}, InvalidParams(errors.New(i18n.Text("--id or --event-id is required")))
		}
		if req.Kind == "restart" {
			id, err := timetools.NewID()
			if err != nil {
				return timetools.Snapshot{}, err
			}
			cmd.NewID = id
		}
	default:
		return timetools.Snapshot{}, InvalidParams(fmt.Errorf(i18n.Text("Unknown time tool command %q"), req.Kind))
	}
	snap, err := timetools.NewStore(e.opts.TimeToolsDir).Update(req.IfRevision, func(s timetools.Snapshot) (timetools.Snapshot, error) {
		return timetools.Apply(s, cmd, e.opts.Now().UnixMilli())
	})
	if err != nil {
		var te *timetools.Error
		if errors.As(err, &te) {
			return timetools.Snapshot{}, Rejected(err)
		}
		return timetools.Snapshot{}, err
	}
	e.publish()
	return snap, nil
}

// RunTimeTools runs the countdown worker in-process and turns completions into
// time_tool_done events. If the standalone launchd worker still holds the
// runner lock, it retries until that worker yields.
func (e *Engine) RunTimeTools(ctx context.Context) {
	worker := timetools.Worker{
		Store:                 timetools.NewStore(e.opts.TimeToolsDir),
		Now:                   e.opts.Now,
		NotificationAvailable: func() bool { return true },
		Send: func(_ context.Context, ev timetools.Event) error {
			label := ev.Label
			if label == "" {
				label = i18n.Text("Timer")
			}
			e.opts.Emit(reminder.Event{
				Type:  reminder.TypeTimeToolDone,
				ID:    ev.ID,
				Title: i18n.Text("Timer complete"),
				Body:  i18n.Text("Timer complete: {0}", label),
			})
			e.publish()
			return nil
		},
	}
	for ctx.Err() == nil {
		if err := worker.Run(ctx); err != nil && timetools.ErrorCode(err) != "conflict" {
			logging.Log(e.opts.LogPath, err.Error())
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
}
