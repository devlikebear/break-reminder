package timetools

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerClaimRecoveryAndDelivery(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Update(nil, func(s Snapshot) (Snapshot, error) { return startTimer(t, 100000), nil })
	if err != nil {
		t.Fatal(err)
	}
	worker := Worker{Store: store, Now: func() time.Time { return time.UnixMilli(110000) }}
	if err = worker.Reconcile(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	worker.Send = func(ctx context.Context, e Event) error { calls++; return errors.New("test failure") }
	if err = worker.DeliverNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	s, _ := store.Load()
	if calls != 1 || s.Events[0].DeliveryState != "failed" {
		t.Fatal(calls, s.Events)
	}
	if err = worker.DeliverNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("automatic retry")
	}
	_, err = store.Update(nil, func(s Snapshot) (Snapshot, error) { s.Events[0].DeliveryState = "claimed"; return s, nil })
	if err != nil {
		t.Fatal(err)
	}
	if err = worker.RecoverClaims(); err != nil {
		t.Fatal(err)
	}
	s, _ = store.Load()
	if s.Events[0].DeliveryState != "unknown" {
		t.Fatal(s.Events)
	}
}
func TestSchedulerAcknowledgedBeforeDeliveryIsSilent(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Update(nil, func(s Snapshot) (Snapshot, error) {
		s = Reconcile(startTimer(t, 100000), 110000)
		return Apply(s, Command{Kind: "acknowledge", ID: s.Events[0].ID}, 110000)
	})
	if err != nil {
		t.Fatal(err)
	}
	worker := Worker{Store: store, Now: func() time.Time { return time.UnixMilli(110000) }, Send: func(context.Context, Event) error { t.Fatal("should not send"); return nil }}
	if err = worker.DeliverNext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestWorkerSingleInstanceAndShutdown(t *testing.T) {
	store := NewStore(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := Worker{Store: store, Interval: 5 * time.Millisecond, NotificationAvailable: func() bool { return false }}
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	limit := time.Now().Add(2 * time.Second)
	for !ReadRuntime(store).Running(time.Now().UnixMilli()) && time.Now().Before(limit) {
		time.Sleep(5 * time.Millisecond)
	}
	if !ReadRuntime(store).Running(time.Now().UnixMilli()) {
		t.Fatal("worker not ready")
	}
	second := Worker{Store: store}
	if err := second.Run(ctx); ErrorCode(err) != "conflict" {
		t.Fatalf("second worker: %v", err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if ReadRuntime(store).Running(time.Now().UnixMilli()) {
		t.Fatal("stopped worker still healthy")
	}
}

func TestWorkerRefreshesHealthAfterWallClockMovesBackward(t *testing.T) {
	store := NewStore(t.TempDir())
	var now atomic.Int64
	now.Store(200000)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	w := Worker{Store: store, Interval: 5 * time.Millisecond, Now: func() time.Time { return time.UnixMilli(now.Load()) }}
	go func() { done <- w.Run(ctx) }()
	defer func() { cancel(); <-done }()
	wait := func(want int64) bool {
		limit := time.Now().Add(time.Second)
		for time.Now().Before(limit) {
			if ReadRuntime(store).Heartbeat == want {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return false
	}
	if !wait(200000) {
		t.Fatal("worker did not start")
	}
	now.Store(100000)
	if !wait(100000) {
		t.Fatal("heartbeat stays in future after clock adjustment")
	}
}
