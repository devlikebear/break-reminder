package engine

import (
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

var meetBase = time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)

func mAt(min int) int64 { return meetBase.Add(time.Duration(min) * time.Minute).Unix() }

func awareConfig() config.Config {
	cfg := testConfig()
	cfg.CalendarAwareBreaks = true
	return cfg
}

func activeState() state.State {
	s := state.New().StartSession(mAt(-60), false)
	s.LastCheck = mAt(-1)
	return s
}

func snapWith(events ...workcontext.Event) workcontext.CalendarSnapshot {
	return workcontext.CalendarSnapshot{UpdatedAt: mAt(-1), Events: events}
}

func mtg(id string, from, to int) workcontext.Event {
	return workcontext.Event{ID: id, Start: mAt(from), End: mAt(to), HasAttendees: true, Availability: "busy"}
}

func TestMeetingPausesUntilItEnds(t *testing.T) {
	r := applyMeeting(awareConfig(), activeState(), snapWith(mtg("m1", 0, 30)), time.Unix(mAt(1), 0))
	s := r.State
	if !s.Paused || s.PauseReason != state.PauseReasonMeeting || s.PauseSource != state.PauseSourceCalendar || s.PauseMeetingID != "m1" || s.PauseUntil != mAt(30) {
		t.Fatalf("state = %+v", s)
	}
	// The existing auto-resume path ends it at PauseUntil.
	if !s.Paused || s.PauseUntil == 0 {
		t.Fatal("pause needs an end")
	}
}

func TestMeetingIgnoredWhenDisabledStaleOrNotRunning(t *testing.T) {
	ev := snapWith(mtg("m1", 0, 30))
	off := awareConfig()
	off.CalendarAwareBreaks = false
	if r := applyMeeting(off, activeState(), ev, time.Unix(mAt(1), 0)); r.State.Paused {
		t.Fatal("disabled: paused")
	}
	stale := ev
	stale.UpdatedAt = mAt(-30)
	if r := applyMeeting(awareConfig(), activeState(), stale, time.Unix(mAt(1), 0)); r.State.Paused {
		t.Fatal("stale snapshot: paused")
	}
	if r := applyMeeting(awareConfig(), state.New(), ev, time.Unix(mAt(1), 0)); r.State.Paused {
		t.Fatal("no running session: paused")
	}
	excluded := awareConfig()
	excludedEv := mtg("m1", 0, 30)
	excludedEv.CalendarID = "shared"
	excluded.CalendarExcludedIDs = []string{"shared"}
	if r := applyMeeting(excluded, activeState(), snapWith(excludedEv), time.Unix(mAt(1), 0)); r.State.Paused {
		t.Fatal("excluded calendar: paused")
	}
}

func TestManualPauseIsLeftAlone(t *testing.T) {
	s := activeState().Pause(mAt(0), state.PauseReasonFocus, 0)
	r := applyMeeting(awareConfig(), s, snapWith(mtg("m1", 0, 30)), time.Unix(mAt(1), 0))
	if r.State.PauseSource != "" || r.State.PauseReason != state.PauseReasonFocus {
		t.Fatalf("manual pause changed: %+v", r.State)
	}
	r = applyMeeting(awareConfig(), s, snapWith(), time.Unix(mAt(1), 0))
	if !r.State.Paused {
		t.Fatal("manual pause must not be resumed by the calendar")
	}
}

func TestCalendarPauseFollowsTheCalendar(t *testing.T) {
	r := applyMeeting(awareConfig(), activeState(), snapWith(mtg("m1", 0, 30)), time.Unix(mAt(1), 0))
	// Meeting extended: the pause end moves.
	r = applyMeeting(awareConfig(), r.State, snapWith(mtg("m1", 0, 45)), time.Unix(mAt(5), 0))
	if r.State.PauseUntil != mAt(45) {
		t.Fatalf("extended: until %d, want %d", r.State.PauseUntil, mAt(45))
	}
	// Meeting deleted: resume right away.
	r = applyMeeting(awareConfig(), r.State, snapWith(), time.Unix(mAt(10), 0))
	if r.State.Paused {
		t.Fatal("deleted meeting: still paused")
	}
	// Feature switched off mid-meeting: resume.
	p := applyMeeting(awareConfig(), activeState(), snapWith(mtg("m2", 0, 30)), time.Unix(mAt(1), 0)).State
	off := awareConfig()
	off.CalendarAwareBreaks = false
	if r := applyMeeting(off, p, snapWith(mtg("m2", 0, 30)), time.Unix(mAt(2), 0)); r.State.Paused {
		t.Fatal("disabled mid-meeting: still paused")
	}
}

func TestManualResumeHoldsForThatMeeting(t *testing.T) {
	p := applyMeeting(awareConfig(), activeState(), snapWith(mtg("m1", 0, 30)), time.Unix(mAt(1), 0)).State
	resumed := resumeByHand(p, mAt(3))
	if resumed.Paused || resumed.MeetingOverrideID != "m1" {
		t.Fatalf("resume = %+v", resumed)
	}
	if r := applyMeeting(awareConfig(), resumed, snapWith(mtg("m1", 0, 30)), time.Unix(mAt(4), 0)); r.State.Paused {
		t.Fatal("re-paused the meeting the user resumed")
	}
	// The next meeting pauses again.
	later := snapWith(mtg("m2", 30, 60))
	later.UpdatedAt = mAt(30)
	if r := applyMeeting(awareConfig(), resumed, later, time.Unix(mAt(31), 0)); !r.State.Paused {
		t.Fatal("next meeting should pause")
	}
}

func TestShortBreakNoticeBeforeMeeting(t *testing.T) {
	s := activeState()
	s.WorkSeconds = 26 * 60
	r := applyMeeting(awareConfig(), s, snapWith(mtg("m1", 4, 30)), time.Unix(mAt(0), 0))
	if len(r.Events) != 1 || r.Events[0].Type != reminder.TypeMeetingSoon || r.State.MeetingNoticeID != "m1" {
		t.Fatalf("notice = %+v / %+v", r.Events, r.State)
	}
	if again := applyMeeting(awareConfig(), r.State, snapWith(mtg("m1", 4, 30)), time.Unix(mAt(1), 0)); len(again.Events) != 0 {
		t.Fatal("notice repeated")
	}
	short := activeState()
	short.WorkSeconds = 10 * 60
	if r := applyMeeting(awareConfig(), short, snapWith(mtg("m1", 4, 30)), time.Unix(mAt(0), 0)); len(r.Events) != 0 {
		t.Fatal("notice after only 10 minutes of work")
	}
	if r := applyMeeting(awareConfig(), s, snapWith(mtg("m1", 8, 30)), time.Unix(mAt(0), 0)); len(r.Events) != 0 {
		t.Fatal("notice 8 minutes early")
	}
}

func TestNoBreakScreenDuringMeeting(t *testing.T) {
	cfg := awareConfig() // 1-minute work blocks
	e, c, r, _ := newTestEngine(t, cfg)
	e.ReportIdle(0) // opens the session
	now := c.Now().Unix()
	if err := e.UpdateCalendar([]workcontext.Event{{ID: "m", Start: now, End: now + 600, HasAttendees: true, Availability: "busy"}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		c.Advance(time.Minute)
		e.ReportIdle(0)
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	for _, typ := range r.types() {
		if typ == reminder.TypeBreakStart {
			t.Fatalf("break started during a meeting: %v", r.types())
		}
	}
	// After the meeting the cycle continues and a break comes due again.
	c.Advance(6 * time.Minute)
	for i := 0; i < 3; i++ {
		c.Advance(time.Minute)
		e.ReportIdle(0)
		_ = e.Tick()
	}
	found := false
	for _, typ := range r.types() {
		found = found || typ == reminder.TypeBreakStart
	}
	if !found {
		t.Fatalf("no break after the meeting: %v", r.types())
	}
}
