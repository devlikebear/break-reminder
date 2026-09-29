package timetools

import (
	"testing"
)

func startTimer(t *testing.T, now int64) Snapshot {
	t.Helper()
	s, err := Apply(NewSnapshot(), Command{Kind: "start", ID: "a", DurationMS: 10000, Label: "리뷰"}, now)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestCountdownPauseResumeAndWallClock(t *testing.T) {
	s := startTimer(t, 100000)
	s, err := Apply(s, Command{Kind: "pause", ID: "a"}, 104000)
	if err != nil || *s.Countdown.RemainingMS != 6000 {
		t.Fatalf("pause: %+v %v", s, err)
	}
	s, err = Apply(s, Command{Kind: "resume", ID: "a"}, 134000)
	if err != nil || *s.Countdown.Deadline != 140000 {
		t.Fatalf("resume: %+v %v", s, err)
	}
	s = Reconcile(s, 140000)
	if s.Countdown.Phase != "completed" || len(s.Events) != 1 {
		t.Fatal(s)
	}
	s = Reconcile(s, 800000)
	if len(s.Events) != 1 {
		t.Fatal("duplicate completion")
	}
}
func TestCountdownCancelReplaceAndStaleID(t *testing.T) {
	s := startTimer(t, 100000)
	if _, err := Apply(s, Command{Kind: "start", ID: "b", DurationMS: 5000}, 101000); ErrorCode(err) != "conflict" {
		t.Fatal(err)
	}
	s, err := Apply(s, Command{Kind: "start", ID: "b", ReplaceID: "a", DurationMS: 5000}, 101000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Apply(s, Command{Kind: "pause", ID: "a"}, 102000); ErrorCode(err) != "not_found" {
		t.Fatal(err)
	}
	s, err = Apply(s, Command{Kind: "cancel", ID: "b"}, 102000)
	if err != nil || s.Countdown.Phase != "canceled" || len(Reconcile(s, 999999).Events) != 0 {
		t.Fatal(s, err)
	}
}
func TestCountdownDeadlineWinsOverCancel(t *testing.T) {
	s, err := Apply(startTimer(t, 100000), Command{Kind: "cancel", ID: "a"}, 110000)
	if err != nil || s.Countdown.Phase != "completed" || len(s.Events) != 1 {
		t.Fatal(s, err)
	}
}
func TestReconcileLateAndAcknowledge(t *testing.T) {
	for _, tt := range []struct {
		now      int64
		delivery string
	}{{410000, "pending"}, {410001, "silent"}} {
		s := Reconcile(startTimer(t, 100000), tt.now)
		if s.Events[0].DeliveryState != tt.delivery {
			t.Fatal(s.Events)
		}
		s, err := Apply(s, Command{Kind: "acknowledge", ID: s.Events[0].ID}, tt.now)
		if err != nil || s.Events[0].AcknowledgedAt == nil || s.Events[0].DeliveryState != "silent" {
			t.Fatal(s.Events, err)
		}
	}
}
func TestCountdownValidation(t *testing.T) {
	for _, c := range []Command{{Kind: "start", ID: "a", DurationMS: 0}, {Kind: "start", ID: "a", DurationMS: 999}, {Kind: "start", ID: "a", DurationMS: 86401000}, {Kind: "start", ID: "a", DurationMS: 1500}, {Kind: "start", ID: "a", DurationMS: 1000, Label: "hello\nworld"}} {
		if _, err := Apply(NewSnapshot(), c, 100000); ErrorCode(err) != "invalid_input" {
			t.Fatal(c, err)
		}
	}
}
func TestRecentAndRestartPreserveCompletion(t *testing.T) {
	s := Reconcile(startTimer(t, 100000), 110000)
	s, err := Apply(s, Command{Kind: "restart", ID: "a", NewID: "b"}, 111000)
	if err != nil || s.Countdown.ID != "b" || len(s.Events) != 1 || len(s.Recent) != 1 {
		t.Fatal(s, err)
	}
}
