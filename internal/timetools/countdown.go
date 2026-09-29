package timetools

import "math"

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
					return original, fail("conflict", "이 알림은 다시 보낼 수 없습니다")
				}
				e.DeliveryState = "pending"
				e.AttemptedAt = ptr(now)
				e.ErrorCode = ""
			}
			return prune(s), nil
		}
		return original, fail("not_found", "완료 기록을 찾을 수 없습니다")
	}
	if cmd.Kind == "start" {
		return start(s, cmd, now)
	}
	c := s.Countdown
	if c == nil || cmd.ID == "" || cmd.ID != c.ID {
		return original, fail("not_found", "이 타이머는 더 이상 현재 타이머가 아닙니다. 새로고침해 주세요")
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
				return original, fail("invalid_input", "시각 범위를 초과했습니다")
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
			return original, fail("conflict", "진행 중인 타이머는 먼저 취소해 주세요")
		}
		return start(s, Command{Kind: "start", ID: cmd.NewID, Label: c.Label, DurationMS: c.DurationMS}, now)
	default:
		return original, fail("invalid_input", "알 수 없는 명령입니다")
	}
	return s, nil
}
func start(s Snapshot, cmd Command, now int64) (Snapshot, error) {
	if cmd.DurationMS < 1000 || cmd.DurationMS > MaxDurationMS || cmd.DurationMS%1000 != 0 || cmd.ID == "" || now > math.MaxInt64-cmd.DurationMS {
		return s, fail("invalid_input", "시간은 1초부터 24시간까지 초 단위로 입력해 주세요")
	}
	label, err := labelValue(cmd.Label)
	if err != nil {
		return s, err
	}
	if c := s.Countdown; c != nil {
		if cmd.ReplaceID != "" && cmd.ReplaceID != c.ID {
			return s, fail("conflict", "교체 대상 타이머가 변경되었습니다")
		}
		if cmd.ID == c.ID {
			return s, fail("conflict", "새 실행에는 새 ID가 필요합니다")
		}
		if (c.Phase == "running" || c.Phase == "paused") && cmd.ReplaceID != c.ID {
			return s, fail("conflict", "실행 중인 타이머가 있습니다. 교체를 확인해 주세요")
		}
	} else if cmd.ReplaceID != "" {
		return s, fail("not_found", "교체할 타이머가 없습니다")
	}
	unread := 0
	for _, e := range s.Events {
		if e.AcknowledgedAt == nil {
			unread++
		}
	}
	if unread >= 100 {
		return s, fail("conflict", "완료 알림을 확인한 뒤 새 타이머를 시작해 주세요")
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
