package workcontext

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

var base = time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)

func at(min int) int64 { return base.Add(time.Duration(min) * time.Minute).Unix() }

func meeting(id string, from, to int) Event {
	return Event{ID: id, Title: id, Start: at(from), End: at(to), HasAttendees: true, Availability: "busy"}
}

func TestIsMeetingFollowsRule(t *testing.T) {
	solo := Event{ID: "solo", Start: at(0), End: at(30), Availability: "busy"}
	allDay := Event{ID: "day", Start: at(-600), End: at(800), AllDay: true, HasAttendees: true, Availability: "busy"}
	free := meeting("free", 0, 30)
	free.Availability = "free"
	m := meeting("m", 0, 30)
	cases := []struct {
		e    Event
		rule string
		want bool
	}{
		{m, config.MeetingRuleAttendees, true},
		{solo, config.MeetingRuleAttendees, false},
		{solo, config.MeetingRuleBusy, true},
		{allDay, config.MeetingRuleBusy, false},
		{free, config.MeetingRuleBusy, false},
	}
	for _, c := range cases {
		if got := IsMeeting(c.e, c.rule); got != c.want {
			t.Errorf("IsMeeting(%s, %s) = %v, want %v", c.e.ID, c.rule, got, c.want)
		}
	}
}

func TestCurrentMeetingBoundariesAndOverlap(t *testing.T) {
	events := []Event{meeting("a", 0, 30), meeting("b", 20, 60)}
	now := func(min int) time.Time { return time.Unix(at(min), 0) }
	if got := CurrentMeeting(events, now(-1), config.MeetingRuleAttendees); got != nil {
		t.Fatalf("before start: %v", got.ID)
	}
	if got := CurrentMeeting(events, now(0), config.MeetingRuleAttendees); got == nil || got.ID != "a" {
		t.Fatalf("at start: %v", got)
	}
	// Overlap: the meeting that keeps you busy longest wins.
	if got := CurrentMeeting(events, now(25), config.MeetingRuleAttendees); got == nil || got.ID != "b" {
		t.Fatalf("overlap: %v", got)
	}
	if got := CurrentMeeting(events, now(60), config.MeetingRuleAttendees); got != nil {
		t.Fatalf("end is exclusive: %v", got.ID)
	}
}

func TestNextEventSkipsAllDayAndFree(t *testing.T) {
	allDay := Event{ID: "day", Start: at(5), End: at(900), AllDay: true}
	free := Event{ID: "free", Start: at(10), End: at(20), Availability: "free"}
	solo := Event{ID: "solo", Start: at(40), End: at(50), Availability: "busy"}
	got := NextEvent([]Event{allDay, free, meeting("m", 90, 120), solo}, time.Unix(at(0), 0))
	if got == nil || got.ID != "solo" {
		t.Fatalf("next = %v", got)
	}
}

func TestFocusWindow(t *testing.T) {
	cfg := config.Default() // pomodoro 25 + 5
	now := time.Unix(at(0), 0)
	// 25+5, 25 = 55 min holds two focus blocks; 52 min holds only one.
	if min, pomos := FocusWindow([]Event{meeting("m", 55, 80)}, now, cfg); min != 55 || pomos != 2 {
		t.Fatalf("55 min -> %d min, %d pomodoros; want 55, 2", min, pomos)
	}
	if min, pomos := FocusWindow([]Event{meeting("m", 52, 80)}, now, cfg); min != 52 || pomos != 1 {
		t.Fatalf("52 min -> %d min, %d pomodoros; want 52, 1", min, pomos)
	}
	if min, pomos := FocusWindow([]Event{meeting("m", 20, 80)}, now, cfg); min != 20 || pomos != 0 {
		t.Fatalf("20 min -> %d, %d", min, pomos)
	}
	if min, pomos := FocusWindow([]Event{meeting("m", -10, 10)}, now, cfg); min != 0 || pomos != 0 {
		t.Fatalf("in a meeting -> %d, %d", min, pomos)
	}
	// Nothing ahead: focus until the detection window closes (22:00 by default).
	if min, _ := FocusWindow(nil, now, cfg); min != 12*60 {
		t.Fatalf("open day -> %d min, want 720", min)
	}
}

func TestFocusWindowAcrossMidnight(t *testing.T) {
	cfg := config.Default()
	cfg.SessionDetectEndHour = 24
	late := time.Date(2026, 10, 20, 23, 30, 0, 0, time.Local)
	next := Event{ID: "early", Start: time.Date(2026, 10, 21, 0, 10, 0, 0, time.Local).Unix(), End: time.Date(2026, 10, 21, 1, 0, 0, 0, time.Local).Unix(), Availability: "busy"}
	if min, _ := FocusWindow([]Event{next}, late, cfg); min != 30 {
		t.Fatalf("the day's window ends at midnight before the event: %d, want 30", min)
	}
}

func TestSnapshotStalenessFilterAndPersistence(t *testing.T) {
	snap := CalendarSnapshot{UpdatedAt: at(0), Events: []Event{meeting("keep", 0, 10), {ID: "drop", CalendarID: "work-holidays"}}}
	if snap.Stale(time.Unix(at(14), 0)) || !snap.Stale(time.Unix(at(16), 0)) {
		t.Fatal("stale after 15 minutes")
	}
	if !(CalendarSnapshot{}).Stale(time.Unix(at(0), 0)) {
		t.Fatal("a never-updated snapshot is stale")
	}
	if got := snap.Visible([]string{"work-holidays"}); len(got) != 1 || got[0].ID != "keep" {
		t.Fatalf("Visible = %v", got)
	}
	path := filepath.Join(t.TempDir(), "calendar-cache.json")
	if err := SaveCalendar(path, snap); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadCalendar(path)
	if err != nil || len(loaded.Events) != 2 || loaded.UpdatedAt != snap.UpdatedAt {
		t.Fatalf("round trip = %+v, %v", loaded, err)
	}
	if missing, err := LoadCalendar(filepath.Join(t.TempDir(), "none.json")); err != nil || missing.UpdatedAt != 0 {
		t.Fatalf("missing cache = %+v, %v", missing, err)
	}
}
