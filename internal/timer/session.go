package timer

import (
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
)

// Session end reasons reported in SessionSummary.
const (
	SessionEndIdle   = "idle"   // the user went away for session_idle_end_min
	SessionEndWindow = "window" // the detection window closed for the day
	SessionEndManual = "manual" // the user ran `break-reminder stop`
)

// SessionSummary describes a work session that just finished.
type SessionSummary struct {
	Start        int64
	End          int64
	WorkSeconds  int
	BreakSeconds int
	Pomodoros    int
	Reason       string
}

// Duration returns the wall-clock span of the session in seconds.
func (s SessionSummary) Duration() int {
	if s.Start <= 0 || s.End <= s.Start {
		return 0
	}
	return int(s.End - s.Start)
}

// EndSessionSummary builds the summary for a session that ends at the given
// time. It is exported so the `stop` command reports the same numbers the
// automatic end reports.
func EndSessionSummary(s state.State, at int64, reason string) SessionSummary {
	return SessionSummary{
		Start:        s.SessionStart,
		End:          at,
		WorkSeconds:  s.TodayWorkSeconds,
		BreakSeconds: s.TodayBreakSeconds,
		Pomodoros:    s.TodayPomodoros,
		Reason:       reason,
	}
}

// evaluateSession applies work-session transitions and reports whether the
// timer should keep ticking. It is pure: every change lands in the returned
// TickResult.
func evaluateSession(cfg config.Config, r TickResult, now time.Time, idleSec int) (TickResult, bool) {
	unix := now.Unix()

	// Remember the most recent moment the user was known to be at the machine.
	if activityAt := unix - int64(idleSec); activityAt > r.State.LastActivity {
		r.State.LastActivity = activityAt
	}

	manuallyStopped := r.State.SessionState == state.SessionStateEnded && r.State.SessionManual

	if !cfg.AutoSessionDetect {
		// Legacy fixed-hours scheduling; explicit start/stop still wins.
		if manuallyStopped {
			return r, false
		}
		if r.State.IsSessionActive() && r.State.SessionManual {
			return r, true
		}
		return r, schedule.IsWorkingTime(cfg, now)
	}

	if r.State.IsSessionActive() {
		if at, reason := sessionEndsAt(cfg, r.State, now, idleSec); reason != "" {
			return endSession(r, at, reason), false
		}
		return r, true
	}

	if manuallyStopped {
		return r, false
	}

	// First activity inside the detection window opens a new session.
	if schedule.InDetectWindow(cfg, now) && idleSec < cfg.IdleThresholdSec {
		r.State = r.State.StartSession(unix, false)
		r.Actions = append(r.Actions, ActionNotifySessionStart)
		r.LogMsg = "Work session started (auto-detected)"
		return r, true
	}

	return r, false
}

// sessionEndsAt reports when and why an active session should end. An empty
// reason means the session continues.
func sessionEndsAt(cfg config.Config, s state.State, now time.Time, idleSec int) (int64, string) {
	unix := now.Unix()

	if idleSec >= cfg.SessionIdleEndSec() {
		endAt := s.LastActivity
		if endAt <= 0 || endAt > unix {
			endAt = unix - int64(idleSec)
		}
		if endAt < s.SessionStart {
			endAt = s.SessionStart
		}
		return endAt, SessionEndIdle
	}

	// Sessions the user opened by hand only end on idle or an explicit stop.
	if !s.SessionManual && now.Hour() >= cfg.SessionDetectEndHour {
		return unix, SessionEndWindow
	}

	return 0, ""
}

func endSession(r TickResult, at int64, reason string) TickResult {
	summary := EndSessionSummary(r.State, at, reason)
	r.State = r.State.EndSession(at, false)
	r.SessionEnded = &summary
	r.Actions = append(r.Actions, ActionNotifySessionEnd)
	r.LogMsg = "Work session ended (" + reason + ")"
	return r
}
