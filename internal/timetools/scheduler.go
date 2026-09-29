package timetools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Runtime struct {
	SchemaVersion         int    `json:"schema_version"`
	Heartbeat             int64  `json:"heartbeat_unix_ms"`
	PID                   int    `json:"pid"`
	Ready                 bool   `json:"ready"`
	NotificationAvailable bool   `json:"notification_available"`
	Error                 string `json:"error,omitempty"`
}

func (r Runtime) Running(now int64) bool {
	return r.SchemaVersion == 1 && r.Ready && r.PID > 0 && now >= r.Heartbeat && now-r.Heartbeat <= 15000 && syscall.Kill(r.PID, 0) == nil
}
func runtimePath(store *Store) string {
	return filepath.Join(filepath.Dir(store.Path), "time-tools-runtime.json")
}
func ReadRuntime(store *Store) Runtime {
	var r Runtime
	data, err := os.ReadFile(runtimePath(store))
	if err == nil {
		_ = json.Unmarshal(data, &r)
	}
	return r
}

type Worker struct {
	Store                 *Store
	Now                   func() time.Time
	Interval              time.Duration
	Send                  func(context.Context, Event) error
	NotificationAvailable func() bool
}

func (w *Worker) now() int64 {
	if w.Now != nil {
		return w.Now().UnixMilli()
	}
	return time.Now().UnixMilli()
}
func (w *Worker) Reconcile() error {
	_, err := w.Store.Update(nil, func(s Snapshot) (Snapshot, error) { return Reconcile(s, w.now()), nil })
	return err
}
func (w *Worker) RecoverClaims() error {
	_, err := w.Store.Update(nil, func(s Snapshot) (Snapshot, error) {
		for i := range s.Events {
			e := &s.Events[i]
			if e.DeliveryState == "claimed" {
				e.DeliveryState = "unknown"
				e.ErrorCode = "interrupted"
			}
		}
		return s, nil
	})
	return err
}
func (w *Worker) DeliverNext(ctx context.Context) error {
	var claimed *Event
	_, err := w.Store.Update(nil, func(s Snapshot) (Snapshot, error) {
		for i := range s.Events {
			e := &s.Events[i]
			if e.DeliveryState != "pending" {
				continue
			}
			if e.AcknowledgedAt != nil || (e.AttemptedAt == nil && w.now()-e.DueAt > 300000) {
				e.DeliveryState = "silent"
				continue
			}
			e.DeliveryState = "claimed"
			e.AttemptedAt = ptr(w.now())
			copy := *e
			claimed = &copy
			break
		}
		return s, nil
	})
	if err != nil || claimed == nil {
		return err
	}
	// An acknowledgement may have arrived after the claim but before send.
	latest, err := w.Store.Load()
	if err != nil {
		return err
	}
	skip := false
	for _, e := range latest.Events {
		if e.ID == claimed.ID && e.AcknowledgedAt != nil {
			skip = true
		}
	}
	var sendErr error
	if !skip {
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if w.Send == nil {
			sendErr = fmt.Errorf("%s", i18n.Text("notification backend unavailable"))
		} else {
			sendErr = w.Send(sendCtx, *claimed)
		}
		cancel()
	}
	_, err = w.Store.Update(nil, func(s Snapshot) (Snapshot, error) {
		for i := range s.Events {
			e := &s.Events[i]
			if e.ID == claimed.ID && e.DeliveryState == "claimed" {
				if skip {
					e.DeliveryState = "silent"
				} else if sendErr != nil {
					e.DeliveryState = "failed"
					e.ErrorCode = "notification_failed"
				} else {
					e.DeliveryState = "sent"
					e.ErrorCode = ""
				}
			}
		}
		return s, nil
	})
	return err
}

// Run owns one worker lock. Sending runs separately from deadline reconciliation.
func (w *Worker) Run(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(w.Store.Path), 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(w.Store.Path+".runner.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fail("conflict", i18n.Text("The time-tools worker is already running"))
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	recoveryErr := w.RecoverClaims()
	available := false
	if w.NotificationAvailable != nil {
		available = w.NotificationAvailable()
	}
	writeHealth := func(ready bool, problem error) error {
		r := Runtime{SchemaVersion: 1, Heartbeat: w.now(), PID: os.Getpid(), Ready: ready, NotificationAvailable: available}
		if problem != nil {
			r.Error = problem.Error()
		}
		data, _ := json.Marshal(r)
		return atomicWrite(runtimePath(w.Store), data)
	}
	defer writeHealth(false, nil)
	interval := w.Interval
	if interval <= 0 {
		interval = time.Second
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var senders sync.WaitGroup
	senders.Add(1)
	go func() {
		defer senders.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				if recoveryErr == nil {
					_ = w.DeliverNext(childCtx)
				}
			}
		}
	}()
	defer func() { cancel(); senders.Wait() }()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lastHealth := int64(0)
	for {
		problem := recoveryErr
		if problem == nil {
			problem = w.Reconcile()
		}
		if lastHealth == 0 || w.now() < lastHealth || w.now()-lastHealth >= 5000 || problem != nil {
			if err := writeHealth(problem == nil, problem); err != nil {
				return err
			}
			lastHealth = w.now()
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
