package timetools

import (
	"github.com/devlikebear/break-reminder/internal/i18n"
	"math"
)

type Command struct {
	Kind       string
	ID         string
	NewID      string
	ReplaceID  string
	Label      string
	DurationMS int64
}

// Reconcile uses a fresh wall-clock timestamp, never a count of elapsed ticks.
func Reconcile(original Snapshot, now int64) Snapshot {
	s := original.clone()
	c := s.Countdown
	if c == nil || c.Phase != "running" || c.Deadline == nil || now < *c.Deadline {
		return s
	}
	due := *c.Deadline
	c.Phase = "completed"
	c.CompletedAt = ptr(due)
	c.RemainingMS = nil
	c.Deadline = nil
	delivery := "pending"
	if now-due > 300000 {
		delivery = "silent"
	}
	eventID := c.ID + ":completed"
	for _, e := range s.Events {
		if e.ID == eventID {
			return s
		}
	}
	s.Events = append(s.Events, Event{ID: eventID, SourceID: c.ID, Kind: "countdown", Label: c.Label, DueAt: due, DeliveryState: delivery})
	return s
}

func Apply(original Snapshot, cmd Command, now int64) (Snapshot, error) {
	s := Reconcile(original, now)
	if cmd.Kind == "acknowledge" || cmd.Kind == "notify-again" {
		for i := range s.Events {
			e := &s.Events[i]
			if e.ID != cmd.ID {
				continue
			}
			if cmd.Kind == "acknowledge" {
				if e.AcknowledgedAt == nil {
					e.AcknowledgedAt = ptr(now)
				}
				if e.DeliveryState == "pending" {
					e.DeliveryState = "silent"
				}
			} else {
				if e.AcknowledgedAt != nil || (e.DeliveryState != "failed" && e.DeliveryState != "unknown") {
					return original, fail("conflict", i18n.Text("This notification cannot be retried"))
				}
				e.DeliveryState = "pending"
				e.AttemptedAt = ptr(now)
				e.ErrorCode = ""
			}
			return prune(s), nil
		}
		return original, fail("not_found", i18n.Text("Completion event not found"))
	}
	if cmd.Kind == "start" {
		return start(s, cmd, now)
	}
	c := s.Countdown
	if c == nil || cmd.ID == "" || cmd.ID != c.ID {
		return original, fail("not_found", i18n.Text("This is no longer the current timer. Refresh and try again"))
	}
	switch cmd.Kind {
	case "pause":
		if c.Phase == "running" {
			c.RemainingMS = ptr(*c.Deadline - now)
			c.Deadline = nil
			c.Phase = "paused"
		}
	case "resume":
		if c.Phase == "paused" {
			if now > math.MaxInt64-*c.RemainingMS {
				return original, fail("invalid_input", i18n.Text("Time is out of range"))
			}
			c.Deadline = ptr(now + *c.RemainingMS)
			c.RemainingMS = nil
			c.Phase = "running"
		}
	case "cancel":
		if c.Phase == "running" || c.Phase == "paused" {
			c.Phase = "canceled"
			c.Deadline = nil
			c.RemainingMS = nil
		}
	case "restart":
		if c.Phase != "completed" && c.Phase != "canceled" {
			return original, fail("conflict", i18n.Text("Cancel the active timer first"))
		}
		return start(s, Command{Kind: "start", ID: cmd.NewID, Label: c.Label, DurationMS: c.DurationMS}, now)
	default:
		return original, fail("invalid_input", i18n.Text("Unknown command"))
	}
	return s, nil
}
func start(s Snapshot, cmd Command, now int64) (Snapshot, error) {
	if cmd.DurationMS < 1000 || cmd.DurationMS > MaxDurationMS || cmd.DurationMS%1000 != 0 || cmd.ID == "" || now > math.MaxInt64-cmd.DurationMS {
		return s, fail("invalid_input", i18n.Text("Enter a duration from 1 second to 24 hours in whole seconds"))
	}
	label, err := labelValue(cmd.Label)
	if err != nil {
		return s, err
	}
	if c := s.Countdown; c != nil {
		if cmd.ReplaceID != "" && cmd.ReplaceID != c.ID {
			return s, fail("conflict", i18n.Text("The timer to replace has changed"))
		}
		if cmd.ID == c.ID {
			return s, fail("conflict", i18n.Text("A new run requires a new ID"))
		}
		if (c.Phase == "running" || c.Phase == "paused") && cmd.ReplaceID != c.ID {
			return s, fail("conflict", i18n.Text("A timer is active. Confirm replacement first"))
		}
	} else if cmd.ReplaceID != "" {
		return s, fail("not_found", i18n.Text("No timer to replace"))
	}
	unread := 0
	for _, e := range s.Events {
		if e.AcknowledgedAt == nil {
			unread++
		}
	}
	if unread >= 100 {
		return s, fail("conflict", i18n.Text("Acknowledge completed timers before starting another"))
	}
	s.Countdown = &Countdown{ID: cmd.ID, Label: label, DurationMS: cmd.DurationMS, Phase: "running", Deadline: ptr(now + cmd.DurationMS), CreatedAt: now}
	recent := []Recent{{Label: label, DurationMS: cmd.DurationMS}}
	for _, r := range s.Recent {
		if r.Label != label || r.DurationMS != cmd.DurationMS {
			recent = append(recent, r)
		}
		if len(recent) == 5 {
			break
		}
	}
	s.Recent = recent
	return s, nil
}
func prune(s Snapshot) Snapshot {
	acknowledged := 0
	out := []Event{}
	for i := len(s.Events) - 1; i >= 0; i-- {
		e := s.Events[i]
		if e.AcknowledgedAt != nil {
			acknowledged++
			if acknowledged > 100 {
				continue
			}
		}
		out = append(out, e)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	s.Events = out
	return s
}
