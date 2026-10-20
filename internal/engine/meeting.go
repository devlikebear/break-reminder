package engine

import (
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// Pre-meeting notice: a meeting this close, after this much continuous work.
const (
	meetingNoticeLead    = 5 * time.Minute
	meetingNoticeWorkSec = 25 * 60
)

type meetingResult struct {
	State  state.State
	Events []reminder.Event
	Log    string
}

// applyMeeting is the calendar-aware break rule, run before each timer tick.
// It only ever touches pauses it started itself (PauseSource=calendar), and
// reuses the regular meeting pause so the paused time counts as neither work
// nor break and PauseUntil auto-resumes when the meeting ends.
func applyMeeting(cfg config.Config, s state.State, snap workcontext.CalendarSnapshot, now time.Time) meetingResult {
	r := meetingResult{State: s}
	calendarPaused := s.Paused && s.PauseSource == state.PauseSourceCalendar

	if !cfg.CalendarAwareBreaks {
		if calendarPaused {
			r.State = s.Resume(now.Unix())
			r.Log = i18n.Text("Calendar-aware breaks are off; resumed the meeting pause")
		}
		return r
	}
	// Without a fresh calendar nothing new is decided; a running calendar
	// pause still ends on its own at PauseUntil.
	if snap.Stale(now) {
		return r
	}

	events := snap.Visible(cfg.CalendarExcludedIDs)
	current := workcontext.CurrentMeeting(events, now, cfg.MeetingRule)

	if s.Paused {
		if !calendarPaused {
			return r
		}
		if current == nil {
			r.State = s.Resume(now.Unix())
			r.Log = i18n.Text("Meeting is no longer on the calendar; timer resumed")
			return r
		}
		r.State.PauseMeetingID = current.ID
		r.State.PauseUntil = current.End
		return r
	}

	if current != nil {
		if current.ID == s.MeetingOverrideID || !schedule.IsActive(cfg, s, now) {
			return r
		}
		p := s.Pause(now.Unix(), state.PauseReasonMeeting, int(current.End-now.Unix()))
		p.PauseSource = state.PauseSourceCalendar
		p.PauseMeetingID = current.ID
		r.State = p
		r.Log = i18n.Text("Paused for a calendar meeting")
		return r
	}

	if next := nextMeeting(events, now, cfg.MeetingRule); next != nil &&
		next.ID != s.MeetingNoticeID &&
		time.Unix(next.Start, 0).Sub(now) <= meetingNoticeLead &&
		s.Mode == "work" && s.WorkSeconds >= meetingNoticeWorkSec &&
		schedule.IsActive(cfg, s, now) {
		r.State.MeetingNoticeID = next.ID
		r.Events = append(r.Events, reminder.Event{
			Type:  reminder.TypeMeetingSoon,
			Title: i18n.Text("Meeting in 5 minutes"),
			Body:  i18n.Text("Stretch and grab some water before it starts~"),
			Sound: "Glass",
		})
	}
	return r
}

func nextMeeting(events []workcontext.Event, now time.Time, rule string) *workcontext.Event {
	var best *workcontext.Event
	for i := range events {
		e := events[i]
		if !workcontext.IsMeeting(e, rule) || e.Start <= now.Unix() {
			continue
		}
		if best == nil || e.Start < best.Start {
			best = &e
		}
	}
	return best
}

// resumeByHand resumes a pause the user ended themselves. Ending a calendar
// pause by hand remembers the meeting so it is not paused again.
func resumeByHand(s state.State, at int64) state.State {
	if s.Paused && s.PauseSource == state.PauseSourceCalendar {
		s.MeetingOverrideID = s.PauseMeetingID
	}
	return s.Resume(at)
}
