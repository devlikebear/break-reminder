# DashboardApp v2 Phase 2: Tabs + Data Visualization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add tabbed navigation (Timer / Stats / Insights-placeholder) to the SwiftUI dashboard, extend history data with hourly work tracking, and build the Stats tab using Swift Charts (stacked bar chart, heatmap, summary cards).

**Architecture:** Extend `DailySummary` Go struct with an `hourly_work` field (backward compatible — missing values default to zero-array). Add hourly accumulation in the Go timer tick. Parse the extended history JSON in Swift via a new `HistoryParser` in HelperCore. Swift Charts (`BarMark`) renders the bar chart; a custom `Grid` + `RoundedRectangle` renders the heatmap.

**Tech Stack:** SwiftUI, Swift Charts, HelperCore, Go (timer/history extension)

**Prerequisites:** Phase 1 complete (SwiftUI migration merged)

---

### Task 1: Extend Go DailySummary with hourly_work field

**Files:**
- Modify: `internal/ai/history.go:9-16`
- Test: `internal/ai/ai_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/ai/ai_test.go` (append at the end):

```go
func TestDailySummaryHourlyWorkPersists(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()
	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	summary := DailySummary{
		Date:       "2026-04-17",
		WorkMin:    280,
		BreakMin:   60,
		Sessions:   7,
		Activities: 3,
		HourlyWork: [24]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 45, 55, 50, 10, 40, 50, 35, 20, 0, 0, 0, 0, 0, 0, 0},
	}
	if err := AppendHistory(summary); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	if history[0].HourlyWork[10] != 55 {
		t.Errorf("HourlyWork[10] = %d, want 55", history[0].HourlyWork[10])
	}
}

func TestDailySummaryBackwardCompatMissingHourly(t *testing.T) {
	origPath := historyPathOverride
	defer func() { historyPathOverride = origPath }()
	historyPathOverride = filepath.Join(t.TempDir(), "history.json")

	// Write legacy JSON without hourly_work field
	legacy := `[{"date":"2026-04-16","work_min":200,"break_min":40,"sessions":4,"activities":2}]`
	if err := os.WriteFile(historyPathOverride, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	for i, v := range history[0].HourlyWork {
		if v != 0 {
			t.Errorf("HourlyWork[%d] = %d, want 0 for legacy data", i, v)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/ -run TestDailySummaryHourlyWork -v`
Expected: FAIL with "unknown field HourlyWork"

- [ ] **Step 3: Add HourlyWork field to DailySummary**

Edit `internal/ai/history.go` — change the struct:

```go
// DailySummary represents one day's usage statistics.
type DailySummary struct {
	Date       string  `json:"date"`
	WorkMin    int     `json:"work_min"`
	BreakMin   int     `json:"break_min"`
	Sessions   int     `json:"sessions"`
	Activities int     `json:"activities"`
	HourlyWork [24]int `json:"hourly_work"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ai/ -v`
Expected: All tests pass including the two new ones

- [ ] **Step 5: Commit**

```bash
git add internal/ai/history.go internal/ai/ai_test.go
git commit -m "feat(history): add hourly_work field to DailySummary"
```

---

### Task 2: Add hourly work tracking to Go State

**Files:**
- Modify: `internal/state/state.go:22-34`
- Test: `internal/state/state_test.go` (create if needed)

- [ ] **Step 1: Check if state test file exists**

Run: `ls internal/state/`

If `state_test.go` exists, note its contents for patterns. If not, we'll create one.

- [ ] **Step 2: Add HourlyWork field to State struct**

Edit `internal/state/state.go` — change the struct:

```go
// State represents the application's current timer state.
type State struct {
	WorkSeconds            int     `json:"work_seconds"`
	Mode                   string  `json:"mode"` // "work" or "break"
	LastCheck              int64   `json:"last_check"`
	BreakStart             int64   `json:"break_start"`
	SnoozeUntil            int64   `json:"snooze_until"`
	Paused                 bool    `json:"paused"`
	PausedAt               int64   `json:"paused_at"`
	TodayWorkSeconds       int     `json:"today_work_seconds"`
	TodayBreakSeconds      int     `json:"today_break_seconds"`
	LastUpdateDate         string  `json:"last_update_date"`
	LastBreakWarningBucket int     `json:"last_break_warning_bucket"`
	HourlyWork             [24]int `json:"hourly_work"`
}
```

- [ ] **Step 3: Update state Load/Save for new field**

Check `internal/state/state.go` for how Load/Save handle fields. If they use key=value parsing (per CLAUDE.md, state uses key=value format), add parse/serialize for HourlyWork as a comma-separated list.

Find the parse/serialize functions (likely `parseLine` and a Save function). Add:

In the parser switch statement, add:
```go
case "HOURLY_WORK":
    parts := strings.Split(val, ",")
    if len(parts) == 24 {
        for i, p := range parts {
            if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
                s.HourlyWork[i] = n
            }
        }
    }
```

In the Save function, add a line:
```go
hourlyParts := make([]string, 24)
for i, v := range s.HourlyWork {
    hourlyParts[i] = strconv.Itoa(v)
}
fmt.Fprintf(w, "HOURLY_WORK=%s\n", strings.Join(hourlyParts, ","))
```

- [ ] **Step 4: Write a round-trip test**

Add to `internal/state/state_test.go` (create file if needed):

```go
package state

import (
	"path/filepath"
	"testing"
)

func TestStateHourlyWorkRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state")

	s := New()
	s.HourlyWork[9] = 600
	s.HourlyWork[14] = 1200

	if err := Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.HourlyWork[9] != 600 {
		t.Errorf("HourlyWork[9] = %d, want 600", loaded.HourlyWork[9])
	}
	if loaded.HourlyWork[14] != 1200 {
		t.Errorf("HourlyWork[14] = %d, want 1200", loaded.HourlyWork[14])
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/state/ -v`
Expected: All tests pass including the round-trip test

- [ ] **Step 6: Commit**

```bash
git add internal/state/
git commit -m "feat(state): add hourly_work tracking to State"
```

---

### Task 3: Accumulate hourly work in timer tick

**Files:**
- Modify: `internal/timer/timer.go:104-159` (tickWork function)
- Test: `internal/timer/timer_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/timer/timer_test.go` (append at end):

```go
func TestTickWorkAccumulatesHourlyWork(t *testing.T) {
	cfg := config.Default()
	s := state.New()
	s.Mode = "work"
	// Simulate 10:30 AM
	now := time.Date(2026, 4, 17, 10, 30, 0, 0, time.Local)
	s.LastCheck = now.Add(-60 * time.Second).Unix()
	s.LastUpdateDate = now.Format("2006-01-02")

	result := Tick(cfg, s, now, 0) // idle = 0, user active
	if result.State.HourlyWork[10] < 60 {
		t.Errorf("HourlyWork[10] = %d, want >= 60 after 60s active work", result.State.HourlyWork[10])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/timer/ -run TestTickWorkAccumulatesHourlyWork -v`
Expected: FAIL (HourlyWork[10] is 0)

- [ ] **Step 3: Implement hourly accumulation**

Edit `internal/timer/timer.go`, in `tickWork`, within the `if idleSec < cfg.IdleThresholdSec` block — add after `r.State.TodayWorkSeconds += elapsed`:

```go
hour := time.Unix(unix, 0).Hour()
if hour >= 0 && hour < 24 {
    r.State.HourlyWork[hour] += elapsed
}
```

Also in the daily reset block (where `TodayWorkSeconds = 0` is set), zero out HourlyWork:

```go
if today != s.LastUpdateDate && s.LastUpdateDate != "" {
    if s.TodayWorkSeconds > 0 || s.TodayBreakSeconds > 0 {
        result.DayEndSummary = &DayEndSummary{
            Date:         s.LastUpdateDate,
            WorkSeconds:  s.TodayWorkSeconds,
            BreakSeconds: s.TodayBreakSeconds,
            HourlyWork:   s.HourlyWork, // preserve for history save
        }
        result.Actions = append(result.Actions, ActionSaveDailyHistory)
    }
    result.State.TodayWorkSeconds = 0
    result.State.TodayBreakSeconds = 0
    result.State.LastUpdateDate = today
    result.State.HourlyWork = [24]int{} // reset for new day
    result.LogMsg = "New day detected! Resetting daily stats."
}
```

- [ ] **Step 4: Update DayEndSummary struct**

Edit `internal/timer/timer.go:25-29`:

```go
// DayEndSummary holds the previous day's stats when a daily reset occurs.
type DayEndSummary struct {
	Date         string
	WorkSeconds  int
	BreakSeconds int
	HourlyWork   [24]int
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/timer/ -v`
Expected: All tests pass including the new one

- [ ] **Step 6: Commit**

```bash
git add internal/timer/
git commit -m "feat(timer): accumulate hourly work seconds per hour bucket"
```

---

### Task 4: Persist HourlyWork to history on day-end

**Files:**
- Find and modify the code that handles `ActionSaveDailyHistory` (search for it)

- [ ] **Step 1: Locate the history save handler**

Run: Grep for `ActionSaveDailyHistory` to find where the action is handled:

Look in `cmd/break-reminder/` and any service/runner files. Example: `cmd/break-reminder/service.go` or similar.

- [ ] **Step 2: Update the DailySummary construction**

Wherever `DailySummary{...}` is constructed from `DayEndSummary`, add the `HourlyWork` field:

```go
summary := ai.DailySummary{
    Date:       dayEnd.Date,
    WorkMin:    dayEnd.WorkSeconds / 60,
    BreakMin:   dayEnd.BreakSeconds / 60,
    // ... existing fields ...
    HourlyWork: convertSecondsToMinutes(dayEnd.HourlyWork),
}
```

Add a helper in the same file:
```go
func convertSecondsToMinutes(seconds [24]int) [24]int {
    var minutes [24]int
    for i, s := range seconds {
        minutes[i] = s / 60
    }
    return minutes
}
```

- [ ] **Step 3: Run all Go tests**

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(history): persist hourly work when saving daily summary"
```

---

### Task 5: Create HistoryParser in HelperCore (Swift)

**Files:**
- Create: `helpers/Sources/HelperCore/HistoryParser.swift`
- Create: `helpers/Tests/HelperCoreTests/HistoryParserTests.swift`

- [ ] **Step 1: Write the failing test**

Create `helpers/Tests/HelperCoreTests/HistoryParserTests.swift`:

```swift
import XCTest
@testable import HelperCore

final class HistoryParserTests: XCTestCase {
    func testParseFullEntry() {
        let json = """
        [{
          "date": "2026-04-17",
          "work_min": 280,
          "break_min": 60,
          "sessions": 7,
          "activities": 3,
          "hourly_work": [0,0,0,0,0,0,0,0,0,45,55,50,10,40,50,35,20,0,0,0,0,0,0,0]
        }]
        """
        let entries = parseHistory(from: json)
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries[0].date, "2026-04-17")
        XCTAssertEqual(entries[0].workMin, 280)
        XCTAssertEqual(entries[0].hourlyWork[10], 55)
    }

    func testParseLegacyEntryWithoutHourly() {
        let json = """
        [{
          "date": "2026-04-16",
          "work_min": 200,
          "break_min": 40,
          "sessions": 4,
          "activities": 2
        }]
        """
        let entries = parseHistory(from: json)
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries[0].hourlyWork, Array(repeating: 0, count: 24))
    }

    func testParseEmptyArray() {
        let entries = parseHistory(from: "[]")
        XCTAssertEqual(entries.count, 0)
    }

    func testParseMalformedReturnsEmpty() {
        let entries = parseHistory(from: "{not json")
        XCTAssertEqual(entries.count, 0)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd helpers && swift test --filter HistoryParserTests 2>&1 | tail -20`
Expected: FAIL with "cannot find parseHistory"

- [ ] **Step 3: Create HistoryParser.swift**

Create `helpers/Sources/HelperCore/HistoryParser.swift`:

```swift
import Foundation

public struct HistoryEntry: Equatable {
    public let date: String
    public let workMin: Int
    public let breakMin: Int
    public let sessions: Int
    public let activities: Int
    public let hourlyWork: [Int]  // 24 elements

    public init(date: String, workMin: Int, breakMin: Int, sessions: Int, activities: Int, hourlyWork: [Int]) {
        self.date = date
        self.workMin = workMin
        self.breakMin = breakMin
        self.sessions = sessions
        self.activities = activities
        self.hourlyWork = hourlyWork
    }
}

public func parseHistory(from json: String) -> [HistoryEntry] {
    guard let data = json.data(using: .utf8),
          let array = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]] else {
        return []
    }

    return array.map { dict in
        let hourly = dict["hourly_work"] as? [Int] ?? Array(repeating: 0, count: 24)
        let normalized = hourly.count == 24 ? hourly : Array(repeating: 0, count: 24)
        return HistoryEntry(
            date: dict["date"] as? String ?? "",
            workMin: dict["work_min"] as? Int ?? 0,
            breakMin: dict["break_min"] as? Int ?? 0,
            sessions: dict["sessions"] as? Int ?? 0,
            activities: dict["activities"] as? Int ?? 0,
            hourlyWork: normalized
        )
    }
}

public func loadHistoryFromDisk() -> [HistoryEntry] {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-history.json")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return [] }
    return parseHistory(from: content)
}
```

- [ ] **Step 4: Run tests**

Run: `cd helpers && swift test --filter HistoryParserTests`
Expected: All 4 tests pass

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/HelperCore/HistoryParser.swift helpers/Tests/HelperCoreTests/HistoryParserTests.swift
git commit -m "feat(helpercore): add HistoryParser for DailySummary JSON"
```

---

### Task 6: Add DashboardTab enum and tab state

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardViewModel.swift`

- [ ] **Step 1: Add tab enum and state to ViewModel**

Edit `helpers/Sources/DashboardApp/DashboardViewModel.swift` — add at the top:

```swift
enum DashboardTab: String, CaseIterable, Identifiable {
    case timer = "타이머"
    case stats = "통계"
    case insights = "인사이트"

    var id: String { rawValue }
}
```

Inside the `DashboardViewModel` class, add:

```swift
@Published var selectedTab: DashboardTab = .timer
@Published var history: [HistoryEntry] = []

func loadHistory() {
    history = loadHistoryFromDisk()
}
```

In `start()`, call `loadHistory()` once, and in `refresh()`, call `loadHistory()` so it stays fresh (history updates on day-end).

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardViewModel.swift
git commit -m "feat(dashboard): add DashboardTab enum and tab state"
```

---

### Task 7: Create TabBarView component

**Files:**
- Create: `helpers/Sources/DashboardApp/TabBarView.swift`

- [ ] **Step 1: Create the tab bar view**

Create `helpers/Sources/DashboardApp/TabBarView.swift`:

```swift
import SwiftUI

struct TabBarView: View {
    @Binding var selectedTab: DashboardTab
    let accentColor: Color

    var body: some View {
        HStack(spacing: 0) {
            ForEach(DashboardTab.allCases) { tab in
                Button(action: { selectedTab = tab }) {
                    VStack(spacing: 6) {
                        Text(tab.rawValue)
                            .font(.system(size: 13, weight: selectedTab == tab ? .semibold : .regular))
                            .foregroundColor(selectedTab == tab ? accentColor : .gray)
                        Rectangle()
                            .fill(selectedTab == tab ? accentColor : Color.clear)
                            .frame(height: 2)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 10)
                }
                .buttonStyle(.plain)
            }
        }
        .background(Color(red: 0.1, green: 0.1, blue: 0.12))
        .overlay(
            Rectangle()
                .fill(Color(white: 0.2))
                .frame(height: 1),
            alignment: .bottom
        )
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/TabBarView.swift
git commit -m "feat(dashboard): add TabBarView component"
```

---

### Task 8: Create StatsTabView with period selector

**Files:**
- Create: `helpers/Sources/DashboardApp/StatsTabView.swift`

- [ ] **Step 1: Create the scaffold with period selector**

Create `helpers/Sources/DashboardApp/StatsTabView.swift`:

```swift
import SwiftUI
import Charts
import HelperCore

enum StatsPeriod: String, CaseIterable, Identifiable {
    case week = "주간"
    case month = "월간"
    case all = "전체"

    var id: String { rawValue }

    var days: Int {
        switch self {
        case .week: return 7
        case .month: return 30
        case .all: return 365
        }
    }
}

struct StatsTabView: View {
    @ObservedObject var vm: DashboardViewModel
    @State private var period: StatsPeriod = .week

    private var filteredHistory: [HistoryEntry] {
        let cutoff = Calendar.current.date(byAdding: .day, value: -period.days, to: Date()) ?? Date()
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let cutoffStr = formatter.string(from: cutoff)
        return vm.history.filter { $0.date >= cutoffStr }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                periodSelector
                // Chart and heatmap added in next tasks
                Text("Loading charts...")
                    .foregroundColor(.gray)
                    .font(.system(size: 12))
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 12)
        }
    }

    private var periodSelector: some View {
        Picker("Period", selection: $period) {
            ForEach(StatsPeriod.allCases) { p in
                Text(p.rawValue).tag(p)
            }
        }
        .pickerStyle(.segmented)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/StatsTabView.swift
git commit -m "feat(dashboard): add StatsTabView scaffold with period selector"
```

---

### Task 9: Add stacked bar chart (Work vs Break per day)

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatsTabView.swift`

- [ ] **Step 1: Add the bar chart section**

Edit `StatsTabView.swift` — replace the `"Loading charts..."` placeholder with a new method. Add this method inside `StatsTabView`:

```swift
private var workBreakChart: some View {
    let workColor = Color(red: 0.3, green: 0.8, blue: 0.5)
    let breakColor = Color(red: 0.4, green: 0.7, blue: 1.0)

    return VStack(alignment: .leading, spacing: 8) {
        Text("작업 / 휴식 시간")
            .font(.system(size: 13, weight: .semibold))
            .foregroundColor(Color(white: 0.9))

        Chart {
            ForEach(filteredHistory, id: \.date) { entry in
                BarMark(
                    x: .value("날짜", shortDate(entry.date)),
                    y: .value("분", entry.workMin)
                )
                .foregroundStyle(workColor)

                BarMark(
                    x: .value("날짜", shortDate(entry.date)),
                    y: .value("분", entry.breakMin)
                )
                .foregroundStyle(breakColor)
            }
        }
        .chartForegroundStyleScale([
            "작업": workColor,
            "휴식": breakColor
        ])
        .frame(height: 140)

        HStack(spacing: 16) {
            HStack(spacing: 4) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(workColor)
                    .frame(width: 8, height: 8)
                Text("작업").font(.system(size: 10)).foregroundColor(.gray)
            }
            HStack(spacing: 4) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(breakColor)
                    .frame(width: 8, height: 8)
                Text("휴식").font(.system(size: 10)).foregroundColor(.gray)
            }
        }
    }
}

private func shortDate(_ iso: String) -> String {
    // Convert "2026-04-17" to "4/17"
    let parts = iso.split(separator: "-")
    guard parts.count == 3 else { return iso }
    return "\(Int(parts[1]) ?? 0)/\(Int(parts[2]) ?? 0)"
}
```

Now update the body's VStack to include it:

```swift
var body: some View {
    ScrollView {
        VStack(alignment: .leading, spacing: 16) {
            periodSelector
            workBreakChart
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/StatsTabView.swift
git commit -m "feat(dashboard): add stacked bar chart for daily work/break"
```

---

### Task 10: Add hourly heatmap

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatsTabView.swift`

- [ ] **Step 1: Add the heatmap component**

Inside `StatsTabView`, add these methods:

```swift
private var heatmapView: some View {
    VStack(alignment: .leading, spacing: 6) {
        Text("시간대별 집중도")
            .font(.system(size: 13, weight: .semibold))
            .foregroundColor(Color(white: 0.9))

        heatmapGrid

        heatmapLegend
    }
}

private var heatmapGrid: some View {
    let hours = Array(9...18)
    let entries = Array(filteredHistory.suffix(7))

    return VStack(alignment: .leading, spacing: 2) {
        // Header row with hour labels
        HStack(spacing: 2) {
            Text("").frame(width: 28)
            ForEach(hours, id: \.self) { hour in
                Text("\(hour)")
                    .font(.system(size: 9))
                    .foregroundColor(.gray)
                    .frame(maxWidth: .infinity)
            }
        }

        // One row per day
        ForEach(entries, id: \.date) { entry in
            HStack(spacing: 2) {
                Text(dayLabel(entry.date))
                    .font(.system(size: 9))
                    .foregroundColor(.gray)
                    .frame(width: 28, alignment: .leading)

                ForEach(hours, id: \.self) { hour in
                    RoundedRectangle(cornerRadius: 2)
                        .fill(heatColor(for: entry.hourlyWork[hour]))
                        .frame(height: 14)
                }
            }
        }
    }
}

private var heatmapLegend: some View {
    HStack(spacing: 4) {
        Text("낮음").font(.system(size: 9)).foregroundColor(.gray)
        ForEach([0, 15, 35, 55], id: \.self) { v in
            RoundedRectangle(cornerRadius: 2)
                .fill(heatColor(for: v))
                .frame(width: 12, height: 8)
        }
        Text("높음").font(.system(size: 9)).foregroundColor(.gray)
    }
}

private func heatColor(for minutes: Int) -> Color {
    switch minutes {
    case 0: return Color(red: 0.145, green: 0.145, blue: 0.157)      // #252528
    case 1..<20: return Color(red: 0.102, green: 0.290, blue: 0.180) // #1a4a2e
    case 20..<45: return Color(red: 0.176, green: 0.478, blue: 0.290) // #2d7a4a
    default: return Color(red: 0.302, green: 0.800, blue: 0.502)     // #4dcc80
    }
}

private func dayLabel(_ iso: String) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    guard let date = formatter.date(from: iso) else { return "" }

    let weekdayFormatter = DateFormatter()
    weekdayFormatter.locale = Locale(identifier: "ko_KR")
    weekdayFormatter.dateFormat = "E"
    return weekdayFormatter.string(from: date)
}
```

Add `heatmapView` to the body VStack:

```swift
var body: some View {
    ScrollView {
        VStack(alignment: .leading, spacing: 16) {
            periodSelector
            workBreakChart
            Divider().background(Color(white: 0.2))
            heatmapView
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/StatsTabView.swift
git commit -m "feat(dashboard): add hourly focus heatmap to stats tab"
```

---

### Task 11: Add summary cards (totals)

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatsTabView.swift`

- [ ] **Step 1: Add summary cards component**

Inside `StatsTabView`, add:

```swift
private var summaryCards: some View {
    let totalWork = filteredHistory.reduce(0) { $0 + $1.workMin }
    let totalBreak = filteredHistory.reduce(0) { $0 + $1.breakMin }
    let total = totalWork + totalBreak
    let ratio = total > 0 ? (totalWork * 100) / total : 0

    return HStack(spacing: 8) {
        summaryCard(label: "\(period.rawValue) 작업", value: formatMinutes(totalWork), color: Color(red: 0.3, green: 0.8, blue: 0.5))
        summaryCard(label: "\(period.rawValue) 휴식", value: formatMinutes(totalBreak), color: Color(red: 0.4, green: 0.7, blue: 1.0))
        summaryCard(label: "작업 비율", value: "\(ratio)%", color: Color(red: 1.0, green: 0.8, blue: 0.4))
    }
}

private func summaryCard(label: String, value: String, color: Color) -> some View {
    VStack(spacing: 4) {
        Text(value)
            .font(.system(size: 16, weight: .semibold))
            .foregroundColor(color)
        Text(label)
            .font(.system(size: 9))
            .foregroundColor(.gray)
    }
    .frame(maxWidth: .infinity)
    .padding(.vertical, 10)
    .background(Color(white: 0.15))
    .cornerRadius(8)
}
```

Add to body VStack:

```swift
var body: some View {
    ScrollView {
        VStack(alignment: .leading, spacing: 16) {
            periodSelector
            workBreakChart
            Divider().background(Color(white: 0.2))
            heatmapView
            Divider().background(Color(white: 0.2))
            summaryCards
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/StatsTabView.swift
git commit -m "feat(dashboard): add weekly summary cards to stats tab"
```

---

### Task 12: Create placeholder InsightsTabView

**Files:**
- Create: `helpers/Sources/DashboardApp/InsightsTabView.swift`

- [ ] **Step 1: Create placeholder view**

Create `helpers/Sources/DashboardApp/InsightsTabView.swift`:

```swift
import SwiftUI

struct InsightsTabView: View {
    var body: some View {
        VStack(spacing: 12) {
            Spacer()
            Image(systemName: "sparkles")
                .font(.system(size: 40))
                .foregroundColor(.gray)
            Text("인사이트는 Phase 3에서 제공됩니다")
                .font(.system(size: 12))
                .foregroundColor(.gray)
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/InsightsTabView.swift
git commit -m "feat(dashboard): add placeholder InsightsTabView"
```

---

### Task 13: Wire tabs into DashboardContentView

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`

- [ ] **Step 1: Update DashboardContentView**

Edit `helpers/Sources/DashboardApp/DashboardAppMain.swift` — replace the `DashboardContentView` struct:

```swift
struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel

    private var accentColor: Color {
        if vm.isPaused { return .yellow }
        return vm.isWork ? Color(red: 0.3, green: 0.8, blue: 0.5) : Color(red: 0.4, green: 0.7, blue: 1.0)
    }

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TabBarView(selectedTab: $vm.selectedTab, accentColor: accentColor)

            Group {
                switch vm.selectedTab {
                case .timer:
                    TimerTabView(vm: vm)
                case .stats:
                    StatsTabView(vm: vm)
                case .insights:
                    InsightsTabView()
                }
            }
        }
    }
}
```

- [ ] **Step 2: Build and test**

Run: `cd helpers && swift build -c release && .build/release/DashboardApp`
Expected: Dashboard appears with:
- Status header at top (always visible)
- Tab bar (타이머 / 통계 / 인사이트)
- Clicking tabs switches the content area
- Stats tab shows bar chart + heatmap + summary cards (may be empty if no history exists)

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardAppMain.swift
git commit -m "feat(dashboard): wire tabs into DashboardContentView"
```

---

### Task 14: Full pipeline verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... && cd helpers && swift test`
Expected: All tests pass (Go and Swift)

- [ ] **Step 2: Build and install**

Run: `make build`
Expected: Both binaries build

- [ ] **Step 3: Manual UI test**

Run: `bin/break-dashboard`

Verify:
- All three tabs switch correctly
- Bar chart renders if `~/.break-reminder-history.json` has data
- Heatmap renders with dummy or real data
- Summary cards show accurate totals
- Period selector (주간/월간/전체) changes filtered data
- Status header stays visible when switching tabs
- Work/Break/Paused color states persist across tabs

- [ ] **Step 4: Seed test data (if no history exists)**

If `~/.break-reminder-history.json` doesn't exist, create test data:

```bash
cat > ~/.break-reminder-history.json << 'EOF'
[
  {"date":"2026-04-11","work_min":180,"break_min":30,"sessions":4,"activities":2,"hourly_work":[0,0,0,0,0,0,0,0,0,45,55,50,0,35,35,40,20,0,0,0,0,0,0,0]},
  {"date":"2026-04-12","work_min":220,"break_min":40,"sessions":5,"activities":3,"hourly_work":[0,0,0,0,0,0,0,0,0,55,55,50,0,45,55,40,30,0,0,0,0,0,0,0]},
  {"date":"2026-04-13","work_min":150,"break_min":25,"sessions":3,"activities":1,"hourly_work":[0,0,0,0,0,0,0,0,0,30,40,35,0,30,40,25,10,0,0,0,0,0,0,0]},
  {"date":"2026-04-14","work_min":260,"break_min":50,"sessions":6,"activities":3,"hourly_work":[0,0,0,0,0,0,0,0,0,55,55,50,0,50,55,45,30,0,0,0,0,0,0,0]},
  {"date":"2026-04-15","work_min":120,"break_min":20,"sessions":3,"activities":1,"hourly_work":[0,0,0,0,0,0,0,0,0,20,30,25,0,20,30,20,10,0,0,0,0,0,0,0]},
  {"date":"2026-04-16","work_min":60,"break_min":10,"sessions":1,"activities":0,"hourly_work":[0,0,0,0,0,0,0,0,0,0,10,15,0,10,15,10,0,0,0,0,0,0,0,0]},
  {"date":"2026-04-17","work_min":30,"break_min":5,"sessions":1,"activities":0,"hourly_work":[0,0,0,0,0,0,0,0,0,0,0,0,0,15,15,0,0,0,0,0,0,0,0,0]}
]
EOF
```

Relaunch and verify charts render.

- [ ] **Step 5: Final commit for any fixes**

```bash
git add -A
git commit -m "fix(dashboard): address Phase 2 integration issues" || true
```

---

## Phase 2 Completion Checklist

- [ ] `go test ./...` passes
- [ ] `cd helpers && swift test` passes
- [ ] `make build` succeeds
- [ ] 3 tabs (타이머/통계/인사이트) all render
- [ ] Stats tab bar chart shows daily work/break
- [ ] Stats tab heatmap shows hourly intensity (9-18)
- [ ] Stats tab summary cards compute weekly totals
- [ ] Period selector filters history correctly
- [ ] Status header stays fixed above tab bar
- [ ] History JSON with `hourly_work` written on day-end
- [ ] Legacy history entries (no `hourly_work`) don't crash — render as blank
