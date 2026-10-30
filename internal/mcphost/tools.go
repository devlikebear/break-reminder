// Package mcphost is the one tool layer for every agent: the built-in agent
// (in process), Claude Code through the claude-code-cli provider, and
// external MCP clients (Claude Desktop/Code over the stdio bridge). Tool
// descriptions are written for the model and passed through unchanged.
//
// The package never imports a model SDK; the engine provides the data
// through Backend.
package mcphost

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// ServerName is how MCP clients name this server; Claude Code prefixes its
// tools with mcp__<ServerName>__.
const ServerName = "break_reminder"

// ServerVersion is reported in the MCP handshake.
const ServerVersion = "0.2.0"

// BusyTitle replaces event titles unless share_event_titles is on.
const BusyTitle = "(busy)"

// Tool names.
var (
	ReadTools   = []string{"get_status", "get_today_calendar", "get_focus_window", "get_app_usage", "get_work_history"}
	ActionTools = []string{"start_pomodoro", "start_timer", "pause", "resume"}
)

// SuggestTool lets the built-in agent propose an action as a button instead
// of running it.
const SuggestTool = "suggest_action"

// TimerView is the timer as a model should see it.
type TimerView struct {
	Phase          string `json:"phase"` // idle | working | on_break | paused
	SessionActive  bool   `json:"session_active"`
	PauseReason    string `json:"pause_reason,omitempty"`
	TimerMode      string `json:"timer_mode"` // classic | pomodoro
	WorkMin        int    `json:"work_min"`
	WorkTargetMin  int    `json:"work_target_min"`
	BreakLeftMin   int    `json:"break_left_min,omitempty"`
	TodayWorkMin   int    `json:"today_work_min"`
	TodayBreakMin  int    `json:"today_break_min"`
	TodayPomodoros int    `json:"today_pomodoros"`
}

// CalendarView is today's calendar as the engine judges it.
type CalendarView struct {
	Events         []workcontext.Event `json:"events"`
	Current        *workcontext.Event  `json:"current,omitempty"`
	Next           *workcontext.Event  `json:"next,omitempty"`
	FocusMinutes   int                 `json:"focus_minutes"`
	FocusPomodoros int                 `json:"focus_pomodoros"`
	Stale          bool                `json:"stale"`
}

// Backend is implemented by the engine.
type Backend interface {
	Timer() (TimerView, error)
	Calendar() (CalendarView, error)
	AppUsage(days int) ([]workcontext.AppUsage, error)
	WorkHistory(days int) ([]ai.DailySummary, error)
	ShareEventTitles() bool
	StartPomodoro() error
	StartTimer(seconds int, label string) error
	Pause(reason string, seconds int) error
	Resume() error
}

// Suggestion is an action the agent proposed for the user to confirm.
type Suggestion struct {
	Action  string `json:"action" jsonschema:"one of start_pomodoro, start_timer, pause, resume"`
	Minutes int    `json:"minutes,omitempty" jsonschema:"timer length or pause length in minutes, when relevant"`
	Label   string `json:"label,omitempty" jsonschema:"timer name, for start_timer"`
	Mode    string `json:"mode,omitempty" jsonschema:"pause mode: meeting, focus or afk"`
	Reason  string `json:"reason,omitempty" jsonschema:"one short sentence shown under the button"`
}

// Host builds MCP servers over a Backend.
type Host struct {
	Backend Backend
}

// ServerOptions picks the tools one server exposes.
type ServerOptions struct {
	// Actions adds the tools that change the timer.
	Actions bool
	// Suggest, when set, adds suggest_action and receives each proposal.
	Suggest func(Suggestion)
}

type noArgs struct{}

type daysArg struct {
	Days int `json:"days,omitempty" jsonschema:"number of days back including today, 1-30 (default 7)"`
}

func (a daysArg) value() (int, error) {
	if a.Days == 0 {
		return 7, nil
	}
	if a.Days < 1 || a.Days > 30 {
		return 0, errors.New(i18n.Text("days must be between 1 and 30"))
	}
	return a.Days, nil
}

type focusOut struct {
	FocusMinutes   int                `json:"focus_minutes"`
	FocusPomodoros int                `json:"focus_pomodoros"`
	Next           *workcontext.Event `json:"next,omitempty"`
	Stale          bool               `json:"stale"`
}

type usageOut struct {
	Days int                    `json:"days"`
	Apps []workcontext.AppUsage `json:"apps"`
}

type historyOut struct {
	Days []ai.DailySummary `json:"days"`
}

type doneOut struct {
	OK bool `json:"ok"`
}

type timerArgs struct {
	Minutes int    `json:"minutes" jsonschema:"countdown length in minutes, 1-1440"`
	Label   string `json:"label,omitempty" jsonschema:"optional timer name"`
}

type pauseArgs struct {
	Mode    string `json:"mode,omitempty" jsonschema:"meeting (default), focus (keeps counting as work) or afk (resets the work cycle)"`
	Minutes int    `json:"minutes,omitempty" jsonschema:"auto-resume after this many minutes; 0 = until resumed"`
}

// Server returns a fresh MCP server with the tools opts allows.
func (h *Host) Server(opts ServerOptions) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: ServerName, Title: "Break Reminder", Version: ServerVersion}, nil)
	h.registerRead(s)
	if opts.Actions {
		h.registerActions(s)
	}
	if opts.Suggest != nil {
		registerSuggest(s, opts.Suggest)
	}
	return s
}

func (h *Host) registerRead(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_status",
		Description: "Current work/break timer: phase (idle, working, on_break, paused), minutes worked in this " +
			"block and the block length, today's totals and pomodoros. Call this first for any question about " +
			"how the user's day or current session is going.",
	}, func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, TimerView, error) {
		v, err := h.Backend.Timer()
		return nil, v, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_today_calendar",
		Description: "Today's and tomorrow's calendar events (times are Unix seconds), the meeting in progress " +
			"and the next event. Titles read \"" + BusyTitle + "\" unless the user chose to share them. " +
			"stale=true means the calendar has not synced in 15 minutes; say so instead of guessing.",
	}, func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, CalendarView, error) {
		v, err := h.Backend.Calendar()
		if err != nil {
			return nil, v, err
		}
		if !h.Backend.ShareEventTitles() {
			v.Events = hideTitles(v.Events)
			v.Current = hideTitle(v.Current)
			v.Next = hideTitle(v.Next)
		}
		return nil, v, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_focus_window",
		Description: "Uninterrupted minutes until the next calendar event (or the end of the work day) and how " +
			"many pomodoro focus blocks fit. Use it to suggest what to do now.",
	}, func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, focusOut, error) {
		v, err := h.Backend.Calendar()
		out := focusOut{FocusMinutes: v.FocusMinutes, FocusPomodoros: v.FocusPomodoros, Next: v.Next, Stale: v.Stale}
		if !h.Backend.ShareEventTitles() {
			out.Next = hideTitle(out.Next)
		}
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_app_usage",
		Description: "Foreground time per app (seconds) summed over the last N days, most used first. Only app " +
			"names are recorded, never window titles. Empty when the user turned app usage off.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in daysArg) (*mcp.CallToolResult, usageOut, error) {
		days, err := in.value()
		if err != nil {
			return nil, usageOut{}, err
		}
		apps, err := h.Backend.AppUsage(days)
		return nil, usageOut{Days: days, Apps: apps}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_work_history",
		Description: "Finished days (oldest first): work and break minutes, hourly work minutes, and the " +
			"day's retrospective when one was written. Use it for trends and comparisons with earlier days.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in daysArg) (*mcp.CallToolResult, historyOut, error) {
		days, err := in.value()
		if err != nil {
			return nil, historyOut{}, err
		}
		hist, err := h.Backend.WorkHistory(days)
		return nil, historyOut{Days: hist}, err
	})
}

func (h *Host) registerActions(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_pomodoro",
		Description: "Switch to the pomodoro timer and start a work session now. Only when the user asked for it.",
	}, func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, doneOut, error) {
		return nil, doneOut{OK: true}, h.Backend.StartPomodoro()
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_timer",
		Description: "Start a countdown timer that runs independently of work and breaks. Only when the user asked for it.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in timerArgs) (*mcp.CallToolResult, doneOut, error) {
		if in.Minutes < 1 || in.Minutes > 1440 {
			return nil, doneOut{}, errors.New(i18n.Text("minutes must be between 1 and 1440"))
		}
		return nil, doneOut{OK: true}, h.Backend.StartTimer(in.Minutes*60, in.Label)
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pause",
		Description: "Pause break reminders. Only when the user asked for it.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in pauseArgs) (*mcp.CallToolResult, doneOut, error) {
		if in.Minutes < 0 {
			return nil, doneOut{}, errors.New(i18n.Text("minutes must not be negative"))
		}
		return nil, doneOut{OK: true}, h.Backend.Pause(in.Mode, in.Minutes*60)
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "resume",
		Description: "Resume paused break reminders. Only when the user asked for it.",
	}, func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, doneOut, error) {
		return nil, doneOut{OK: true}, h.Backend.Resume()
	})
}

func registerSuggest(s *mcp.Server, suggest func(Suggestion)) {
	mcp.AddTool(s, &mcp.Tool{
		Name: SuggestTool,
		Description: "Offer the user a one-click action (shown as a button under your answer) instead of doing it. " +
			"Use it when an action would help but the user did not ask you to act. Nothing runs until they click.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Suggestion) (*mcp.CallToolResult, doneOut, error) {
		switch in.Action {
		case "start_pomodoro", "resume":
		case "start_timer":
			if in.Minutes < 1 || in.Minutes > 1440 {
				return nil, doneOut{}, errors.New(i18n.Text("minutes must be between 1 and 1440"))
			}
		case "pause":
			if in.Minutes < 0 {
				return nil, doneOut{}, errors.New(i18n.Text("minutes must not be negative"))
			}
		default:
			return nil, doneOut{}, fmt.Errorf(i18n.Text("unknown action %q"), in.Action)
		}
		suggest(in)
		return nil, doneOut{OK: true}, nil
	})
}

func hideTitle(e *workcontext.Event) *workcontext.Event {
	if e == nil {
		return nil
	}
	c := *e
	c.Title = BusyTitle
	return &c
}

func hideTitles(events []workcontext.Event) []workcontext.Event {
	out := make([]workcontext.Event, len(events))
	for i, e := range events {
		e.Title = BusyTitle
		out[i] = e
	}
	return out
}
