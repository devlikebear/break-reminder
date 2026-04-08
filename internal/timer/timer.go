package timer

import (
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

// Action represents something the timer wants the caller to do.
type Action int

const (
	ActionNone Action = iota
	ActionNotifyBreakTime
	ActionNotifyBreakOver
	ActionNotifyFiveMinWarning
	ActionNotifyStillOnBreak
	ActionSpeakBreakTime
	ActionSpeakBreakOver
	ActionSaveDailyHistory
)

// DayEndSummary holds the previous day's stats when a daily reset occurs.
type DayEndSummary struct {
	Date         string
	WorkSeconds  int
	BreakSeconds int
}

// TickResult is the outcome of a single timer tick.
type TickResult struct {
	State         state.State
	Actions       []Action
	LogMsg        string
	DayEndSummary *DayEndSummary // non-nil when daily reset triggers history save
}

// Tick computes the next state given config, current state, current time, and idle seconds.
func Tick(cfg config.Config, s state.State, now time.Time, idleSec int) TickResult {
	result := TickResult{State: s}
	unix := now.Unix()

	// Daily reset
	today := now.Format("2006-01-02")
	if today != s.LastUpdateDate && s.LastUpdateDate != "" {
		// Save previous day's stats before resetting
		if s.TodayWorkSeconds > 0 || s.TodayBreakSeconds > 0 {
			result.DayEndSummary = &DayEndSummary{
				Date:         s.LastUpdateDate,
				WorkSeconds:  s.TodayWorkSeconds,
				BreakSeconds: s.TodayBreakSeconds,
			}
			result.Actions = append(result.Actions, ActionSaveDailyHistory)
		}
		result.State.TodayWorkSeconds = 0
		result.State.TodayBreakSeconds = 0
		result.State.LastUpdateDate = today
		result.LogMsg = "New day detected! Resetting daily stats."
	} else if s.LastUpdateDate == "" {
		result.State.LastUpdateDate = today
	}

	elapsed := int(unix - s.LastCheck)

	// Reset if too much time has passed (computer restart, sleep, etc.)
	if elapsed > 3600 {
		result.State.WorkSeconds = 0
		result.State.Mode = "work"
		result.State.LastCheck = unix
		result.LogMsg = "Long gap detected, resetting..."
		return result
	}

	// If elapsed exceeds check interval significantly, the computer was likely
	// asleep or the app wasn't running — don't count that time as work.
	maxExpectedElapsed := cfg.CheckIntervalSec * 3 // allow some slack (e.g. 180s for 60s interval)
	if maxExpectedElapsed < 300 {
		maxExpectedElapsed = 300
	}
	if elapsed > maxExpectedElapsed {
		result.State.LastCheck = unix
		result.LogMsg = "Gap detected (" + itoa(elapsed) + "s), skipping as idle"
		return result
	}

	result.State.LastCheck = unix

	switch s.Mode {
	case "work":
		result = tickWork(cfg, result, elapsed, idleSec, unix)
	case "break":
		result = tickBreak(cfg, result, elapsed, idleSec, unix)
	}

	return result
}

func tickWork(cfg config.Config, r TickResult, elapsed, idleSec int, unix int64) TickResult {
	workDur := cfg.WorkDurationSec()

	if idleSec < cfg.IdleThresholdSec {
		// User is active
		r.State.WorkSeconds += elapsed
		r.State.TodayWorkSeconds += elapsed

		workMin := r.State.WorkSeconds / 60
		remainMin := (workDur - r.State.WorkSeconds) / 60
		r.LogMsg = "Working... " + itoa(workMin) + "min elapsed (" + itoa(remainMin) + "min remaining)"

		// Break time!
		if r.State.WorkSeconds >= workDur {
			r.LogMsg = "Break time triggered!"
			r.Actions = append(r.Actions, ActionNotifyBreakTime)
			if cfg.TTSEnabled {
				r.Actions = append(r.Actions, ActionSpeakBreakTime)
			}
			r.State = r.State.EnterBreak(unix)
			return r
		}

		// 5-minute warning
		warningStart := workDur - 5*60
		warningEnd := warningStart + 60
		if r.State.WorkSeconds >= warningStart && r.State.WorkSeconds < warningEnd {
			r.Actions = append(r.Actions, ActionNotifyFiveMinWarning)
		}
	} else {
		// User is idle
		if idleSec > cfg.NaturalBreakSec {
			r.LogMsg = "Natural break detected (idle " + itoa(idleSec) + "s), resetting work time"
			r.State.WorkSeconds = 0
		}
	}

	return r
}

func tickBreak(cfg config.Config, r TickResult, elapsed, idleSec int, unix int64) TickResult {
	breakDur := cfg.BreakDurationSec()

	r.State.TodayBreakSeconds += elapsed
	breakElapsed := int(unix - r.State.BreakStart)
	breakRemaining := (breakDur - breakElapsed) / 60

	r.LogMsg = "Break mode... " + itoa(breakRemaining) + "min remaining"

	// Warn if the user is still active during break, but only once per 2-minute bucket.
	if breakElapsed < breakDur && idleSec < cfg.IdleThresholdSec {
		bucket := breakElapsed / 120
		if bucket > 0 && bucket > r.State.LastBreakWarningBucket {
			r.Actions = append(r.Actions, ActionNotifyStillOnBreak)
			r.State.LastBreakWarningBucket = bucket
		}
	}

	// Break is over
	if breakElapsed >= breakDur {
		r.LogMsg = "Break finished, back to work mode"
		r.Actions = append(r.Actions, ActionNotifyBreakOver)
		if cfg.TTSEnabled {
			r.Actions = append(r.Actions, ActionSpeakBreakOver)
		}
		r.State.Mode = "work"
		r.State.WorkSeconds = 0
		r.State.LastBreakWarningBucket = 0
	}

	return r
}

func itoa(n int) string {
	if n < 0 {
		return "0"
	}
	// Simple int to string without importing strconv
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
