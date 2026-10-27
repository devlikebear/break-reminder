// Package reminder turns timer actions into user-facing reminder events.
//
// The events describe what to show or say; they never touch the OS. The
// launchd `check` path executes them with the Go notifier/speaker, and the
// engine forwards them to the menu bar app, which executes them natively.
package reminder

import (
	"fmt"
	"strings"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/timer"
)

// Event types.
const (
	TypeBreakStart     = "break_start"
	TypeBreakOver      = "break_over"
	TypeFiveMinWarning = "five_min_warning"
	TypeStillOnBreak   = "still_on_break"
	TypeSessionStart   = "session_start"
	TypeSessionEnd     = "session_end"
	TypeSpeak          = "speak"
	TypeTimeToolDone   = "time_tool_done"
	TypeMeetingSoon    = "meeting_soon"
	TypeBriefing       = "briefing"
	TypeRetro          = "retro"
)

// Event is one thing the user should see or hear. A non-empty Title means
// "post a notification"; break_start carries its notification as a fallback
// for when no overlay is shown.
type Event struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Sound string `json:"sound,omitempty"`

	// speak
	Text  string `json:"text,omitempty"`
	Voice string `json:"voice,omitempty"`

	// break_start
	ScreenMode    string `json:"screen_mode,omitempty"` // block | notify | ask
	BreakSec      int    `json:"break_sec,omitempty"`
	BreakStart    int64  `json:"break_start,omitempty"`
	TodayWorkMin  int    `json:"today_work_min,omitempty"`
	TodayBreakMin int    `json:"today_break_min,omitempty"`

	// time_tool_done
	ID string `json:"id,omitempty"`
}

// Events translates the actions of one tick into reminder events, in order.
func Events(cfg config.Config, r timer.TickResult) []Event {
	s := r.State
	workMin := cfg.EffectiveWorkMin()
	var out []Event
	for _, a := range r.Actions {
		switch a {
		case timer.ActionNotifyBreakTime:
			breakSec := cfg.EffectiveBreakSec(s.PomodoroCount)
			out = append(out, Event{
				Type:          TypeBreakStart,
				Title:         i18n.Text("Break Time!"),
				Body:          fmt.Sprintf(i18n.Text("%d minutes complete! Take a %d-minute break~"), workMin, breakSec/60),
				Sound:         "Blow",
				ScreenMode:    cfg.BreakScreenMode,
				BreakSec:      breakSec,
				BreakStart:    s.BreakStart,
				TodayWorkMin:  s.TodayWorkSeconds / 60,
				TodayBreakMin: s.TodayBreakSeconds / 60,
			})
		case timer.ActionNotifyBreakOver:
			out = append(out, Event{Type: TypeBreakOver, Title: i18n.Text("Break Over!"), Body: fmt.Sprintf(i18n.Text("Back to work! %d-minute timer started~"), workMin), Sound: "Hero"})
		case timer.ActionNotifyFiveMinWarning:
			out = append(out, Event{Type: TypeFiveMinWarning, Title: i18n.Text("5 minutes left"), Body: i18n.Text("Break time coming up~")})
		case timer.ActionNotifyStillOnBreak:
			out = append(out, Event{Type: TypeStillOnBreak, Title: i18n.Text("Still on break!"), Body: i18n.Text("Keep resting!")})
		case timer.ActionNotifySessionStart:
			out = append(out, Event{Type: TypeSessionStart, Title: i18n.Text("Work session started"), Body: fmt.Sprintf(i18n.Text("Timer is running — first break in %d minutes."), workMin), Sound: "Submarine"})
		case timer.ActionNotifySessionEnd:
			out = append(out, Event{Type: TypeSessionEnd, Title: i18n.Text("Work session ended"), Body: SessionEndMessage(r.SessionEnded), Sound: "Glass"})
		case timer.ActionSpeakBreakTime:
			out = append(out, Event{Type: TypeSpeak, Voice: cfg.Voice, Text: fmt.Sprintf(i18n.Text("Time for a break! You've been working for %d minutes."), workMin)})
		case timer.ActionSpeakBreakOver:
			out = append(out, Event{Type: TypeSpeak, Voice: cfg.Voice, Text: i18n.Text("Break time is over! Let's get back to work!")})
		}
	}
	return out
}

// SessionEndMessage renders the notification body for a finished work session.
func SessionEndMessage(sum *timer.SessionSummary) string {
	if sum == nil {
		return i18n.Text("Timer stopped. See you next session!")
	}
	parts := []string{
		fmt.Sprintf(i18n.Text("Work %s"), FormatMinutes(sum.WorkSeconds/60)),
		fmt.Sprintf(i18n.Text("Break %s"), FormatMinutes(sum.BreakSeconds/60)),
	}
	if sum.Pomodoros > 0 {
		parts = append(parts, fmt.Sprintf(i18n.Text("%d pomodoro%s"), sum.Pomodoros, PluralS(sum.Pomodoros)))
	}
	return i18n.Text("Today · ") + strings.Join(parts, " · ")
}

// FormatMinutes renders minutes as "1h 5m", "2h" or "45m".
func FormatMinutes(min int) string {
	if min >= 60 {
		h, m := min/60, min%60
		if m > 0 {
			return fmt.Sprintf(i18n.Text("%dh %dm"), h, m)
		}
		return fmt.Sprintf(i18n.Text("%dh"), h)
	}
	return fmt.Sprintf(i18n.Text("%dm"), min)
}

// PluralS returns the English plural suffix ("" in Korean).
func PluralS(n int) string {
	if n == 1 || i18n.Current() == "ko" {
		return ""
	}
	return "s"
}
