package timer

import (
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func TestTick(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	tests := []struct {
		name        string
		state       state.State
		idleSec     int
		wantMode    string
		wantActions []Action
	}{
		{
			name: "work mode, active, accumulates time",
			state: state.State{
				Mode:           "work",
				WorkSeconds:    600, // 10 min
				LastCheck:      now.Add(-60 * time.Second).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:  5,
			wantMode: "work",
		},
		{
			name: "work mode, reaches 50min, switches to break",
			state: state.State{
				Mode:           "work",
				WorkSeconds:    49*60 + 30, // 49.5 min
				LastCheck:      now.Add(-60 * time.Second).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:     5,
			wantMode:    "break",
			wantActions: []Action{ActionNotifyBreakTime, ActionSpeakBreakTime},
		},
		{
			name: "work mode, 5-min warning",
			state: state.State{
				Mode:           "work",
				WorkSeconds:    44 * 60, // 44 min, next tick at 45 min
				LastCheck:      now.Add(-60 * time.Second).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:     5,
			wantMode:    "work",
			wantActions: []Action{ActionNotifyFiveMinWarning},
		},
		{
			name: "work mode, idle natural break resets",
			state: state.State{
				Mode:           "work",
				WorkSeconds:    1800,
				LastCheck:      now.Add(-60 * time.Second).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:  600, // 10 min idle > 5 min threshold
			wantMode: "work",
		},
		{
			name: "break mode, break finished",
			state: state.State{
				Mode:           "break",
				BreakStart:     now.Add(-11 * time.Minute).Unix(),
				LastCheck:      now.Add(-60 * time.Second).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:     5,
			wantMode:    "work",
			wantActions: []Action{ActionNotifyBreakOver, ActionSpeakBreakOver},
		},
		{
			name: "long gap resets to work",
			state: state.State{
				Mode:           "break",
				WorkSeconds:    1000,
				LastCheck:      now.Add(-2 * time.Hour).Unix(),
				LastUpdateDate: now.Format("2006-01-02"),
			},
			idleSec:  0,
			wantMode: "work",
		},
		{
			name: "daily reset on new day",
			state: state.State{
				Mode:              "work",
				WorkSeconds:       600,
				TodayWorkSeconds:  5000,
				TodayBreakSeconds: 1000,
				LastCheck:         now.Add(-60 * time.Second).Unix(),
				LastUpdateDate:    "2025-01-14",
			},
			idleSec:     5,
			wantMode:    "work",
			wantActions: []Action{ActionSaveDailyHistory},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Tick(cfg, tt.state, now, tt.idleSec)

			if result.State.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", result.State.Mode, tt.wantMode)
			}

			if tt.wantActions != nil {
				if len(result.Actions) != len(tt.wantActions) {
					t.Errorf("actions = %v, want %v", result.Actions, tt.wantActions)
				} else {
					for i, a := range tt.wantActions {
						if result.Actions[i] != a {
							t.Errorf("action[%d] = %v, want %v", i, result.Actions[i], a)
						}
					}
				}
			}

			if tt.name == "daily reset on new day" {
				if result.State.TodayWorkSeconds != 60 {
					// Should have reset to 0 then added elapsed
					// Actually: reset to 0, then tickWork adds elapsed (60)
				}
				if result.State.LastUpdateDate != now.Format("2006-01-02") {
					t.Errorf("LastUpdateDate = %q, want %q", result.State.LastUpdateDate, now.Format("2006-01-02"))
				}
			}

			if tt.name == "work mode, idle natural break resets" {
				if result.State.WorkSeconds != 0 {
					t.Errorf("WorkSeconds = %d, want 0 (natural break reset)", result.State.WorkSeconds)
				}
			}

			if tt.name == "daily reset on new day" {
				if result.DayEndSummary == nil {
					t.Fatal("DayEndSummary should be set on daily reset")
				}
				if result.DayEndSummary.Date != "2025-01-14" {
					t.Errorf("DayEndSummary.Date = %q, want %q", result.DayEndSummary.Date, "2025-01-14")
				}
				if result.DayEndSummary.WorkSeconds != 5000 {
					t.Errorf("DayEndSummary.WorkSeconds = %d, want 5000", result.DayEndSummary.WorkSeconds)
				}
				if result.DayEndSummary.BreakSeconds != 1000 {
					t.Errorf("DayEndSummary.BreakSeconds = %d, want 1000", result.DayEndSummary.BreakSeconds)
				}
			}
		})
	}
}

func TestMediumGapSkippedAsIdle(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	s := state.State{
		Mode:              "work",
		WorkSeconds:       600,
		TodayWorkSeconds:  600,
		TodayBreakSeconds: 0,
		LastCheck:         now.Add(-10 * time.Minute).Unix(), // 600s gap
		LastUpdateDate:    now.Format("2006-01-02"),
	}

	result := Tick(cfg, s, now, 5)

	// Should NOT accumulate 600s of work time
	if result.State.TodayWorkSeconds != 600 {
		t.Errorf("TodayWorkSeconds = %d, want 600 (gap should be skipped)", result.State.TodayWorkSeconds)
	}
	if result.State.WorkSeconds != 600 {
		t.Errorf("WorkSeconds = %d, want 600 (gap should not add work)", result.State.WorkSeconds)
	}
}

func TestDailyResetNoHistoryWhenEmpty(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	s := state.State{
		Mode:              "work",
		WorkSeconds:       600,
		TodayWorkSeconds:  0,
		TodayBreakSeconds: 0,
		LastCheck:         now.Add(-60 * time.Second).Unix(),
		LastUpdateDate:    "2025-01-14",
	}

	result := Tick(cfg, s, now, 5)

	if result.DayEndSummary != nil {
		t.Error("DayEndSummary should be nil when previous day had zero work/break")
	}

	hasHistoryAction := false
	for _, a := range result.Actions {
		if a == ActionSaveDailyHistory {
			hasHistoryAction = true
		}
	}
	if hasHistoryAction {
		t.Error("should not emit ActionSaveDailyHistory when previous day had no data")
	}
}

func TestBreakWarningRequiresActiveUser(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 10, 0, 0, time.Local)

	s := state.State{
		Mode:                   "break",
		BreakStart:             now.Add(-3 * time.Minute).Unix(),
		LastCheck:              now.Add(-60 * time.Second).Unix(),
		LastUpdateDate:         now.Format("2006-01-02"),
		LastBreakWarningBucket: 0,
	}

	result := Tick(cfg, s, now, cfg.IdleThresholdSec+30)

	for _, action := range result.Actions {
		if action == ActionNotifyStillOnBreak {
			t.Fatal("expected no active-break warning while user is idle")
		}
	}
	if result.State.LastBreakWarningBucket != 0 {
		t.Fatalf("LastBreakWarningBucket = %d, want 0", result.State.LastBreakWarningBucket)
	}
}

func TestBreakWarningOnlyOncePerBucket(t *testing.T) {
	cfg := config.Default()
	start := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)
	now := start.Add(100 * time.Second)

	s := state.State{
		Mode:                   "break",
		BreakStart:             start.Unix(),
		LastCheck:              now.Add(-30 * time.Second).Unix(),
		LastUpdateDate:         start.Format("2006-01-02"),
		LastBreakWarningBucket: 1,
	}

	result := Tick(cfg, s, now, 5)

	for _, action := range result.Actions {
		if action == ActionNotifyStillOnBreak {
			t.Fatal("expected no duplicate active-break warning in same bucket")
		}
	}
	if result.State.LastBreakWarningBucket != 1 {
		t.Fatalf("LastBreakWarningBucket = %d, want 1", result.State.LastBreakWarningBucket)
	}
}

func TestBreakWarningAdvancesOnNewBucket(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 4, 5, 0, time.Local)

	s := state.State{
		Mode:                   "break",
		BreakStart:             now.Add(-(4*time.Minute + 5*time.Second)).Unix(),
		LastCheck:              now.Add(-60 * time.Second).Unix(),
		LastUpdateDate:         now.Format("2006-01-02"),
		LastBreakWarningBucket: 1,
	}

	result := Tick(cfg, s, now, 5)

	found := false
	for _, action := range result.Actions {
		if action == ActionNotifyStillOnBreak {
			found = true
		}
	}
	if !found {
		t.Fatal("expected an active-break warning when entering a new 2-minute bucket")
	}
	if result.State.LastBreakWarningBucket != 2 {
		t.Fatalf("LastBreakWarningBucket = %d, want 2", result.State.LastBreakWarningBucket)
	}
}

func TestTickPausedSkipsCounting(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 20, 0, 0, time.Local)

	s := state.State{
		Mode:             "work",
		WorkSeconds:      25 * 60,
		TodayWorkSeconds: 2 * 3600,
		LastCheck:        now.Add(-15 * time.Minute).Unix(),
		Paused:           true,
		PausedAt:         now.Add(-10 * time.Minute).Unix(),
		LastUpdateDate:   now.Format("2006-01-02"),
	}

	result := Tick(cfg, s, now, 0)

	if result.State.WorkSeconds != s.WorkSeconds {
		t.Fatalf("WorkSeconds = %d, want %d", result.State.WorkSeconds, s.WorkSeconds)
	}
	if result.State.TodayWorkSeconds != s.TodayWorkSeconds {
		t.Fatalf("TodayWorkSeconds = %d, want %d", result.State.TodayWorkSeconds, s.TodayWorkSeconds)
	}
	if result.State.LastCheck != s.LastCheck {
		t.Fatalf("LastCheck = %d, want %d", result.State.LastCheck, s.LastCheck)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("Actions = %v, want none", result.Actions)
	}
}

func TestWorkTickAtIdleThresholdDoesNotAccumulate(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	s := state.State{
		Mode:              "work",
		WorkSeconds:       600,
		TodayWorkSeconds:  1200,
		LastCheck:         now.Add(-60 * time.Second).Unix(),
		LastUpdateDate:    now.Format("2006-01-02"),
		TodayBreakSeconds: 300,
	}

	result := Tick(cfg, s, now, cfg.IdleThresholdSec)

	if result.State.WorkSeconds != s.WorkSeconds {
		t.Fatalf("WorkSeconds = %d, want %d", result.State.WorkSeconds, s.WorkSeconds)
	}
	if result.State.TodayWorkSeconds != s.TodayWorkSeconds {
		t.Fatalf("TodayWorkSeconds = %d, want %d", result.State.TodayWorkSeconds, s.TodayWorkSeconds)
	}
	for _, action := range result.Actions {
		if action == ActionNotifyFiveMinWarning || action == ActionNotifyBreakTime {
			t.Fatalf("unexpected action at idle threshold: %v", action)
		}
	}
}

func TestWorkTickContinuesAccumulatingDuringSnoozeWithoutTriggeringBreak(t *testing.T) {
	cfg := config.Default()
	start := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	snoozed, err := (state.State{
		Mode:           "break",
		BreakStart:     start.Add(-2 * time.Minute).Unix(),
		LastCheck:      start.Unix(),
		LastUpdateDate: start.Format("2006-01-02"),
	}).SnoozeBreak(start, 5*time.Minute)
	if err != nil {
		t.Fatalf("SnoozeBreak() error = %v", err)
	}

	duringSnooze := Tick(cfg, snoozed, start.Add(2*time.Minute), 5)
	if duringSnooze.State.WorkSeconds != 120 {
		t.Fatalf("WorkSeconds during snooze = %d, want 120", duringSnooze.State.WorkSeconds)
	}
	if duringSnooze.State.TodayWorkSeconds != 120 {
		t.Fatalf("TodayWorkSeconds during snooze = %d, want 120", duringSnooze.State.TodayWorkSeconds)
	}
	if duringSnooze.State.Mode != "work" {
		t.Fatalf("Mode during snooze = %q, want work", duringSnooze.State.Mode)
	}
	if duringSnooze.State.SnoozeUntil != snoozed.SnoozeUntil {
		t.Fatalf("SnoozeUntil during snooze = %d, want %d", duringSnooze.State.SnoozeUntil, snoozed.SnoozeUntil)
	}
	for _, action := range duringSnooze.Actions {
		if action == ActionNotifyBreakTime || action == ActionNotifyFiveMinWarning {
			t.Fatalf("unexpected action during snooze: %v", action)
		}
	}

	afterSnooze := Tick(cfg, duringSnooze.State, start.Add(6*time.Minute), 5)
	if afterSnooze.State.Mode != "break" {
		t.Fatalf("Mode after snooze = %q, want break", afterSnooze.State.Mode)
	}
	if afterSnooze.State.BreakStart != start.Add(6*time.Minute).Unix() {
		t.Fatalf("BreakStart after snooze = %d, want %d", afterSnooze.State.BreakStart, start.Add(6*time.Minute).Unix())
	}
	if afterSnooze.State.SnoozeUntil != 0 {
		t.Fatalf("SnoozeUntil after snooze = %d, want 0", afterSnooze.State.SnoozeUntil)
	}
}

func TestWorkTickDoesNotExpireSnoozeWhilePaused(t *testing.T) {
	cfg := config.Default()
	start := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	snoozed, err := (state.State{
		Mode:           "break",
		BreakStart:     start.Add(-2 * time.Minute).Unix(),
		LastCheck:      start.Unix(),
		LastUpdateDate: start.Format("2006-01-02"),
	}).SnoozeBreak(start, 5*time.Minute)
	if err != nil {
		t.Fatalf("SnoozeBreak() error = %v", err)
	}

	paused := snoozed.Pause(start.Add(30 * time.Second).Unix())
	resumed := paused.Resume(start.Add(6 * time.Minute).Unix())

	if resumed.SnoozeUntil != start.Add(10*time.Minute + 30*time.Second).Unix() {
		t.Fatalf("SnoozeUntil after resume = %d, want %d", resumed.SnoozeUntil, start.Add(10*time.Minute+30*time.Second).Unix())
	}

	afterResume := Tick(cfg, resumed, start.Add(6*time.Minute+30*time.Second), 5)
	if afterResume.State.Mode != "work" {
		t.Fatalf("Mode after resuming mid-snooze = %q, want work", afterResume.State.Mode)
	}
	for _, action := range afterResume.Actions {
		if action == ActionNotifyBreakTime {
			t.Fatalf("unexpected break notification while snooze should still be frozen: %v", action)
		}
	}
}

func TestNaturalBreakClearsPendingSnooze(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 4, 5, 0, time.Local)

	s := state.State{
		Mode:              "work",
		WorkSeconds:       45 * 60,
		LastCheck:         now.Add(-60 * time.Second).Unix(),
		SnoozeUntil:       now.Add(2 * time.Minute).Unix(),
		LastUpdateDate:    now.Format("2006-01-02"),
		TodayWorkSeconds:  45 * 60,
		TodayBreakSeconds: 300,
	}

	result := Tick(cfg, s, now, cfg.NaturalBreakSec+1)
	if result.State.WorkSeconds != 0 {
		t.Fatalf("WorkSeconds = %d, want 0", result.State.WorkSeconds)
	}
	if result.State.SnoozeUntil != 0 {
		t.Fatalf("SnoozeUntil = %d, want 0", result.State.SnoozeUntil)
	}
	for _, action := range result.Actions {
		if action == ActionNotifyBreakTime {
			t.Fatalf("unexpected break notification after natural break: %v", action)
		}
	}
}

func TestBreakWarningSuppressedAtIdleThresholdBoundary(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 15, 10, 4, 5, 0, time.Local)

	s := state.State{
		Mode:                   "break",
		BreakStart:             now.Add(-(4*time.Minute + 5*time.Second)).Unix(),
		LastCheck:              now.Add(-60 * time.Second).Unix(),
		LastUpdateDate:         now.Format("2006-01-02"),
		LastBreakWarningBucket: 1,
	}

	result := Tick(cfg, s, now, cfg.IdleThresholdSec)

	for _, action := range result.Actions {
		if action == ActionNotifyStillOnBreak {
			t.Fatal("expected no active-break warning when idle time is exactly at threshold")
		}
	}
	if result.State.LastBreakWarningBucket != 1 {
		t.Fatalf("LastBreakWarningBucket = %d, want 1", result.State.LastBreakWarningBucket)
	}
}

func TestShortBreakWarningRespectsGracePeriod(t *testing.T) {
	cfg := config.Default()
	cfg.BreakDurationMin = 1
	start := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	t.Run("before grace period", func(t *testing.T) {
		now := start.Add(20 * time.Second)
		s := state.State{
			Mode:                   "break",
			BreakStart:             start.Unix(),
			LastCheck:              now.Add(-10 * time.Second).Unix(),
			LastUpdateDate:         now.Format("2006-01-02"),
			LastBreakWarningBucket: 0,
		}

		result := Tick(cfg, s, now, 5)
		for _, action := range result.Actions {
			if action == ActionNotifyStillOnBreak {
				t.Fatal("expected no warning before the fixed grace period elapses")
			}
		}
	})

	t.Run("warns after grace period", func(t *testing.T) {
		now := start.Add(45 * time.Second)
		s := state.State{
			Mode:                   "break",
			BreakStart:             start.Unix(),
			LastCheck:              now.Add(-15 * time.Second).Unix(),
			LastUpdateDate:         now.Format("2006-01-02"),
			LastBreakWarningBucket: 0,
		}

		result := Tick(cfg, s, now, 5)
		found := false
		for _, action := range result.Actions {
			if action == ActionNotifyStillOnBreak {
				found = true
			}
		}
		if !found {
			t.Fatal("expected a warning during a 1-minute break after the grace period")
		}
	})
}

func TestDailyResetWhileStillOnBreak(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 16, 0, 1, 0, 0, time.Local)

	s := state.State{
		Mode:                   "break",
		BreakStart:             time.Date(2025, 1, 15, 23, 58, 0, 0, time.Local).Unix(),
		LastCheck:              now.Add(-60 * time.Second).Unix(),
		TodayWorkSeconds:       5000,
		TodayBreakSeconds:      900,
		LastUpdateDate:         "2025-01-15",
		LastBreakWarningBucket: 1,
	}

	result := Tick(cfg, s, now, cfg.IdleThresholdSec+30)

	if result.DayEndSummary == nil {
		t.Fatal("expected DayEndSummary on rollover during break")
	}
	if result.DayEndSummary.Date != "2025-01-15" {
		t.Fatalf("DayEndSummary.Date = %q, want 2025-01-15", result.DayEndSummary.Date)
	}
	if result.DayEndSummary.WorkSeconds != 5000 || result.DayEndSummary.BreakSeconds != 900 {
		t.Fatalf("DayEndSummary = %+v, want prior-day totals", *result.DayEndSummary)
	}
	if result.State.Mode != "break" {
		t.Fatalf("Mode = %q, want break", result.State.Mode)
	}
	if result.State.LastUpdateDate != "2025-01-16" {
		t.Fatalf("LastUpdateDate = %q, want 2025-01-16", result.State.LastUpdateDate)
	}
	if result.State.TodayWorkSeconds != 0 {
		t.Fatalf("TodayWorkSeconds = %d, want 0", result.State.TodayWorkSeconds)
	}
	if result.State.TodayBreakSeconds != 60 {
		t.Fatalf("TodayBreakSeconds = %d, want 60", result.State.TodayBreakSeconds)
	}
}

func TestTickWorkAccumulatesHourlyWork(t *testing.T) {
	cfg := config.Default()
	s := state.New()
	s.Mode = "work"
	// Simulate 10:30 AM local time
	now := time.Date(2026, 4, 17, 10, 30, 0, 0, time.Local)
	s.LastCheck = now.Add(-60 * time.Second).Unix()
	s.LastUpdateDate = now.Format("2006-01-02")

	result := Tick(cfg, s, now, 0) // idle = 0 → user active
	if result.State.HourlyWork[10] < 60 {
		t.Errorf("HourlyWork[10] = %d, want >= 60 after 60s active work", result.State.HourlyWork[10])
	}
	// Other hours should be untouched
	for i, v := range result.State.HourlyWork {
		if i == 10 {
			continue
		}
		if v != 0 {
			t.Errorf("HourlyWork[%d] = %d, want 0 (only hour 10 should accumulate)", i, v)
		}
	}
}

func TestTickIdleDoesNotAccumulateHourly(t *testing.T) {
	cfg := config.Default()
	s := state.New()
	s.Mode = "work"
	now := time.Date(2026, 4, 17, 10, 30, 0, 0, time.Local)
	s.LastCheck = now.Add(-60 * time.Second).Unix()
	s.LastUpdateDate = now.Format("2006-01-02")

	// idle >= threshold → user inactive → no hourly accum
	result := Tick(cfg, s, now, cfg.IdleThresholdSec+1)
	for i, v := range result.State.HourlyWork {
		if v != 0 {
			t.Errorf("HourlyWork[%d] = %d, want 0 when user idle", i, v)
		}
	}
}

func TestTickDailyResetClearsHourlyAndPreservesInSummary(t *testing.T) {
	cfg := config.Default()
	s := state.New()
	s.Mode = "work"
	s.LastUpdateDate = "2025-01-14"
	s.TodayWorkSeconds = 1800
	s.TodayBreakSeconds = 600
	s.HourlyWork[10] = 600
	s.HourlyWork[14] = 1200

	// Next day at 9:00 AM
	now := time.Date(2025, 1, 15, 9, 0, 0, 0, time.Local)
	s.LastCheck = now.Add(-60 * time.Second).Unix()

	result := Tick(cfg, s, now, 0)

	if result.DayEndSummary == nil {
		t.Fatal("expected DayEndSummary on daily reset")
	}
	if result.DayEndSummary.HourlyWork[10] != 600 {
		t.Errorf("DayEndSummary.HourlyWork[10] = %d, want 600", result.DayEndSummary.HourlyWork[10])
	}
	if result.DayEndSummary.HourlyWork[14] != 1200 {
		t.Errorf("DayEndSummary.HourlyWork[14] = %d, want 1200", result.DayEndSummary.HourlyWork[14])
	}
	// State should be reset
	for i, v := range result.State.HourlyWork {
		// The tick may have accumulated into the new day's hour[9] bucket
		if i == 9 {
			continue
		}
		if v != 0 {
			t.Errorf("result.State.HourlyWork[%d] = %d after reset, want 0", i, v)
		}
	}
}

func TestTickPausedOverMidnightOnlyRollsDailyTotals(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2025, 1, 16, 0, 5, 0, 0, time.Local)

	s := state.State{
		Mode:              "work",
		WorkSeconds:       1800,
		TodayWorkSeconds:  7200,
		TodayBreakSeconds: 900,
		LastCheck:         time.Date(2025, 1, 15, 23, 55, 0, 0, time.Local).Unix(),
		Paused:            true,
		PausedAt:          time.Date(2025, 1, 15, 23, 58, 0, 0, time.Local).Unix(),
		LastUpdateDate:    "2025-01-15",
	}

	result := Tick(cfg, s, now, 0)

	if result.DayEndSummary == nil {
		t.Fatal("expected DayEndSummary on rollover while paused")
	}
	if result.DayEndSummary.WorkSeconds != 7200 || result.DayEndSummary.BreakSeconds != 900 {
		t.Fatalf("DayEndSummary = %+v, want prior-day totals", *result.DayEndSummary)
	}
	if result.State.TodayWorkSeconds != 0 || result.State.TodayBreakSeconds != 0 {
		t.Fatalf("rolled-over totals = (%d,%d), want 0,0", result.State.TodayWorkSeconds, result.State.TodayBreakSeconds)
	}
	if result.State.LastCheck != s.LastCheck {
		t.Fatalf("LastCheck = %d, want %d", result.State.LastCheck, s.LastCheck)
	}
	if !result.State.Paused {
		t.Fatal("paused state should remain paused")
	}
}
