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
