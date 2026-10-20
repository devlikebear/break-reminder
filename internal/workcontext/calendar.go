// Package workcontext judges the user's working context from data the app
// collects: calendar events and per-app foreground time. Every decision is a
// pure function of its inputs; the engine owns timing and storage.
package workcontext

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

// StaleAfter is how long a calendar snapshot is trusted without an update.
const StaleAfter = 15 * time.Minute

// Event is one calendar event as sent by the app (EventKit). Times are Unix seconds.
type Event struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Start        int64  `json:"start"`
	End          int64  `json:"end"`
	AllDay       bool   `json:"all_day"`
	HasAttendees bool   `json:"has_attendees"`
	Availability string `json:"availability"` // busy | free | tentative | unavailable | ""
	CalendarID   string `json:"calendar_id"`
}

// CalendarSnapshot is the latest event list and when it arrived.
type CalendarSnapshot struct {
	Events    []Event `json:"events"`
	UpdatedAt int64   `json:"updated_at"`
}

// Stale reports whether the snapshot is too old to act on.
func (s CalendarSnapshot) Stale(now time.Time) bool {
	return s.UpdatedAt == 0 || now.Sub(time.Unix(s.UpdatedAt, 0)) > StaleAfter
}

// Visible drops events of calendars the user excluded.
func (s CalendarSnapshot) Visible(excluded []string) []Event {
	skip := map[string]bool{}
	for _, id := range excluded {
		skip[id] = true
	}
	out := make([]Event, 0, len(s.Events))
	for _, e := range s.Events {
		if !skip[e.CalendarID] {
			out = append(out, e)
		}
	}
	return out
}

// blocks reports whether an event occupies time: timed and not marked free.
func blocks(e Event) bool {
	return !e.AllDay && e.Availability != "free" && e.End > e.Start
}

// IsMeeting applies the meeting_rule to one event.
func IsMeeting(e Event, rule string) bool {
	if !blocks(e) {
		return false
	}
	if rule == config.MeetingRuleBusy {
		return true
	}
	return e.HasAttendees
}

// CurrentMeeting returns the meeting in progress at now. When meetings
// overlap, the one ending last wins so a pause covers all of them.
func CurrentMeeting(events []Event, now time.Time, rule string) *Event {
	t := now.Unix()
	var best *Event
	for i := range events {
		e := events[i]
		if !IsMeeting(e, rule) || t < e.Start || t >= e.End {
			continue
		}
		if best == nil || e.End > best.End {
			best = &e
		}
	}
	return best
}

// NextEvent returns the next timed, non-free event starting after now.
func NextEvent(events []Event, now time.Time) *Event {
	t := now.Unix()
	var best *Event
	for i := range events {
		e := events[i]
		if !blocks(e) || e.Start <= t {
			continue
		}
		if best == nil || e.Start < best.Start {
			best = &e
		}
	}
	return best
}

// FocusWindow returns the uninterrupted minutes until the next occupied
// event (or the end of today's detection window) and how many pomodoros fit.
// A pomodoro fits when its focus block fits; the trailing break may run into
// the event.
func FocusWindow(events []Event, now time.Time, cfg config.Config) (minutes, pomodoros int) {
	t := now.Unix()
	for _, e := range events {
		if blocks(e) && e.Start <= t && t < e.End {
			return 0, 0
		}
	}
	y, m, d := now.Date()
	end := time.Date(y, m, d, cfg.SessionDetectEndHour, 0, 0, 0, now.Location()).Unix()
	if next := NextEvent(events, now); next != nil && next.Start < end {
		end = next.Start
	}
	if end <= t {
		return 0, 0
	}
	minutes = int(end-t) / 60
	work, rest := cfg.PomodoroWorkMin, cfg.PomodoroBreakMin
	if work <= 0 {
		return minutes, 0
	}
	cycle := work + rest
	pomodoros = minutes / cycle
	if minutes-pomodoros*cycle >= work {
		pomodoros++
	}
	return minutes, pomodoros
}

// SortEvents orders events by start time for display.
func SortEvents(events []Event) {
	sort.SliceStable(events, func(i, j int) bool { return events[i].Start < events[j].Start })
}

// LoadCalendar reads the cached snapshot; a missing file is an empty snapshot.
func LoadCalendar(path string) (CalendarSnapshot, error) {
	var snap CalendarSnapshot
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	err = json.Unmarshal(data, &snap)
	return snap, err
}

// SaveCalendar writes the snapshot atomically, readable by the owner only.
func SaveCalendar(path string, snap CalendarSnapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
