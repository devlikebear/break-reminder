package timer

import (
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

// 2025-01-15 is a Wednesday, 2025-01-19 a Sunday.
func at(hour, min int) time.Time {
	return time.Date(2025, 1, 15, hour, min, 0, 0, time.Local)
}

func hasAction(actions []Action, want Action) bool {
	for _, a := range actions {
		if a == want {
			return true
		}
	}
	return false
}

func idleState(now time.Time) state.State {
	return state.State{
		Mode:           "work",
		LastCheck:      now.Add(-60 * time.Second).Unix(),
		LastUpdateDate: now.Format("2006-01-02"),
	}
}

func activeState(now time.Time) state.State {
	s := idleState(now)
	s.SessionState = state.SessionStateActive
	s.SessionStart = now.Add(-2 * time.Hour).Unix()
	s.LastActivity = now.Unix()
	return s
}

func TestSessionAutoStartsOnFirstActivity(t *testing.T) {
	cfg := config.Default()
	now := at(8, 0) // before work_start_hour but inside the detection window

	result := Tick(cfg, idleState(now), now, 5)

	if !result.State.IsSessionActive() {
		t.Fatalf("SessionState = %q, want active", result.State.SessionState)
	}
	if result.State.SessionStart != now.Unix() {
		t.Errorf("SessionStart = %d, want %d", result.State.SessionStart, now.Unix())
	}
	if result.State.SessionManual {
		t.Error("auto-detected session should not be marked manual")
	}
	if !hasAction(result.Actions, ActionNotifySessionStart) {
		t.Errorf("actions = %v, want a session start notification", result.Actions)
	}
}

func TestSessionDoesNotStartOutsideDetectWindow(t *testing.T) {
	cfg := config.Default()
	now := at(6, 30) // session_detect_start_hour defaults to 7

	result := Tick(cfg, idleState(now), now, 0)

	if result.State.IsSessionActive() {
		t.Error("session should not start before the detection window opens")
	}
	if len(result.Actions) != 0 {
		t.Errorf("actions = %v, want none", result.Actions)
	}
}

func TestSessionDoesNotStartOnNonWorkingDay(t *testing.T) {
	cfg := config.Default()
	sunday := time.Date(2025, 1, 19, 10, 0, 0, 0, time.Local)

	result := Tick(cfg, idleState(sunday), sunday, 0)

	if result.State.IsSessionActive() {
		t.Error("session should not start on a non-working day")
	}
}

func TestSessionDoesNotStartWhileUserIsAway(t *testing.T) {
	cfg := config.Default()
	now := at(9, 0)

	result := Tick(cfg, idleState(now), now, cfg.IdleThresholdSec+10)

	if result.State.IsSessionActive() {
		t.Error("session should wait for real activity")
	}
}

func TestSessionAutoEndsAfterLongIdle(t *testing.T) {
	cfg := config.Default()
	now := at(13, 0)
	s := activeState(now)
	s.LastActivity = now.Add(-40 * time.Minute).Unix()
	s.TodayWorkSeconds = 3600
	s.TodayBreakSeconds = 600
	s.TodayPomodoros = 2

	result := Tick(cfg, s, now, 40*60)

	if result.State.SessionState != state.SessionStateEnded {
		t.Fatalf("SessionState = %q, want ended", result.State.SessionState)
	}
	if result.State.SessionManual {
		t.Error("an auto-detected end must not set the manual stop flag")
	}
	if !hasAction(result.Actions, ActionNotifySessionEnd) {
		t.Errorf("actions = %v, want a session end notification", result.Actions)
	}
	if result.SessionEnded == nil {
		t.Fatal("SessionEnded summary is missing")
	}
	if result.SessionEnded.End != s.LastActivity {
		t.Errorf("summary End = %d, want the last activity %d", result.SessionEnded.End, s.LastActivity)
	}
	if result.SessionEnded.Reason != SessionEndIdle {
		t.Errorf("summary Reason = %q, want %q", result.SessionEnded.Reason, SessionEndIdle)
	}
	if result.SessionEnded.WorkSeconds != 3600 || result.SessionEnded.BreakSeconds != 600 {
		t.Errorf("summary totals = %d/%d, want 3600/600", result.SessionEnded.WorkSeconds, result.SessionEnded.BreakSeconds)
	}
	if result.SessionEnded.Pomodoros != 2 {
		t.Errorf("summary Pomodoros = %d, want 2", result.SessionEnded.Pomodoros)
	}
}

func TestSessionEndsWhenDetectWindowCloses(t *testing.T) {
	cfg := config.Default()
	now := at(22, 5) // session_detect_end_hour defaults to 22

	result := Tick(cfg, activeState(now), now, 5)

	if result.State.SessionState != state.SessionStateEnded {
		t.Fatalf("SessionState = %q, want ended", result.State.SessionState)
	}
	if result.SessionEnded == nil || result.SessionEnded.Reason != SessionEndWindow {
		t.Fatalf("SessionEnded = %+v, want reason %q", result.SessionEnded, SessionEndWindow)
	}
}

func TestManualSessionIgnoresDetectWindowClose(t *testing.T) {
	cfg := config.Default()
	now := at(23, 30)
	s := activeState(now)
	s.SessionManual = true

	result := Tick(cfg, s, now, 5)

	if !result.State.IsSessionActive() {
		t.Fatal("a manually started session must keep running past the detection window")
	}
}

func TestManualSessionStillEndsOnLongIdle(t *testing.T) {
	cfg := config.Default()
	now := at(15, 0)
	s := activeState(now)
	s.SessionManual = true
	s.LastActivity = now.Add(-45 * time.Minute).Unix()

	result := Tick(cfg, s, now, 45*60)

	if result.State.SessionState != state.SessionStateEnded {
		t.Fatalf("SessionState = %q, want ended", result.State.SessionState)
	}
}

func TestManualStopBlocksAutoRestartSameDay(t *testing.T) {
	cfg := config.Default()
	now := at(14, 0)
	s := idleState(now)
	s.SessionState = state.SessionStateEnded
	s.SessionManual = true
	s.SessionEnd = now.Add(-30 * time.Minute).Unix()

	result := Tick(cfg, s, now, 0)

	if result.State.IsSessionActive() {
		t.Error("a manual stop must hold until the next day or an explicit start")
	}
	if hasAction(result.Actions, ActionNotifySessionStart) {
		t.Error("no session start notification expected after a manual stop")
	}
}

func TestAutoEndedSessionRestartsOnReturn(t *testing.T) {
	cfg := config.Default()
	now := at(14, 0)
	s := idleState(now)
	s.SessionState = state.SessionStateEnded
	s.SessionEnd = now.Add(-45 * time.Minute).Unix()

	result := Tick(cfg, s, now, 0)

	if !result.State.IsSessionActive() {
		t.Error("an automatically ended session should reopen when the user returns")
	}
}

func TestDailyResetClearsManualStop(t *testing.T) {
	cfg := config.Default()
	now := at(9, 0)
	s := idleState(now)
	s.LastUpdateDate = "2025-01-14"
	s.SessionState = state.SessionStateEnded
	s.SessionManual = true
	s.TodayWorkSeconds = 7200
	s.TodayPomodoros = 6

	result := Tick(cfg, s, now, 5)

	if !result.State.IsSessionActive() {
		t.Fatal("the new day should let automatic detection start a session again")
	}
	if result.State.TodayPomodoros != 0 {
		t.Errorf("TodayPomodoros = %d, want 0 after the daily reset", result.State.TodayPomodoros)
	}
}

func TestLegacySchedulingGatesOnWorkingHours(t *testing.T) {
	cfg := config.Default()
	cfg.AutoSessionDetect = false

	evening := at(20, 0)
	s := idleState(evening)
	s.WorkSeconds = 600
	result := Tick(cfg, s, evening, 5)
	if result.State.WorkSeconds != 600 {
		t.Errorf("WorkSeconds = %d, want no accrual outside working hours", result.State.WorkSeconds)
	}
	if result.State.IsSessionActive() {
		t.Error("legacy scheduling should not open sessions on its own")
	}

	midday := at(11, 0)
	s = idleState(midday)
	s.WorkSeconds = 600
	result = Tick(cfg, s, midday, 5)
	if result.State.WorkSeconds != 660 {
		t.Errorf("WorkSeconds = %d, want 660 during working hours", result.State.WorkSeconds)
	}
}

func TestLegacySchedulingHonoursManualControls(t *testing.T) {
	cfg := config.Default()
	cfg.AutoSessionDetect = false

	// Manual start keeps the timer running outside working hours.
	evening := at(21, 0)
	started := idleState(evening).StartSession(evening.Add(-10*time.Minute).Unix(), true)
	started.LastCheck = evening.Add(-60 * time.Second).Unix()
	result := Tick(cfg, started, evening, 5)
	if result.State.WorkSeconds != 60 {
		t.Errorf("WorkSeconds = %d, want 60 for a manually started session", result.State.WorkSeconds)
	}

	// Manual stop keeps it off inside working hours.
	midday := at(11, 0)
	stopped := idleState(midday)
	stopped.WorkSeconds = 300
	stopped = stopped.EndSession(midday.Add(-5*time.Minute).Unix(), true)
	stopped.LastCheck = midday.Add(-60 * time.Second).Unix()
	result = Tick(cfg, stopped, midday, 5)
	if result.State.WorkSeconds != 0 {
		t.Errorf("WorkSeconds = %d, want no accrual after a manual stop", result.State.WorkSeconds)
	}
}

func TestPausedStateIsNotTouchedBySessionDetection(t *testing.T) {
	cfg := config.Default()
	now := at(10, 0)
	s := idleState(now)
	s.Paused = true
	s.PausedAt = now.Add(-10 * time.Minute).Unix()
	s.PauseReason = state.PauseReasonMeeting

	result := Tick(cfg, s, now, 0)

	if result.State.IsSessionActive() {
		t.Error("a paused timer should not open a session")
	}
	if !result.State.Paused {
		t.Error("pause must survive the tick")
	}
}
