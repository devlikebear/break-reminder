# DashboardApp v2 Phase 3: AI Summary — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate AI-powered daily reports and pattern insights using the existing `ai.Client` (claude/codex CLI), save to `~/.break-reminder-insights.json`, and render them in the Insights tab with refresh/copy actions.

**Architecture:** Add an `insights` package in Go that builds prompts from recent history and parses AI responses into structured patterns. Add a CLI subcommand `break-reminder insights --refresh` that forces regeneration. Auto-trigger generation once per day during the tick loop when near work-end time. Swift reads the insights JSON and renders it; a refresh button shells out to `break-reminder insights --refresh`.

**Tech Stack:** Go (ai/insights packages, cobra CLI), SwiftUI, HelperCore

**Prerequisites:** Phase 2 complete (tabs + history data)

---

### Task 1: Create Go insights package with data structures

**Files:**
- Create: `internal/insights/insights.go`
- Create: `internal/insights/insights_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/insights/insights_test.go`:

```go
package insights

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	pathOverride = filepath.Join(t.TempDir(), "nope.json")

	result, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for missing file, got %v", result)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	pathOverride = filepath.Join(t.TempDir(), "insights.json")

	report := &Report{
		GeneratedAt: "2026-04-17T17:30:00+09:00",
		DailyReport: "오늘 4시간 20분 작업",
		Patterns: []Pattern{
			{Type: "warning", Title: "오후 슬럼프", Description: "D", Suggestion: "S"},
		},
	}

	if err := Save(report); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil report")
	}
	if loaded.DailyReport != "오늘 4시간 20분 작업" {
		t.Errorf("DailyReport mismatch: %q", loaded.DailyReport)
	}
	if len(loaded.Patterns) != 1 {
		t.Errorf("Patterns count = %d, want 1", len(loaded.Patterns))
	}
}

func TestSaveCreatesMissingDirectory(t *testing.T) {
	origPath := pathOverride
	defer func() { pathOverride = origPath }()
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	pathOverride = filepath.Join(dir, "insights.json")

	report := &Report{GeneratedAt: "2026-04-17T00:00:00Z"}
	if err := Save(report); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(pathOverride); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/insights/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Create insights.go**

Create `internal/insights/insights.go`:

```go
package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Pattern represents a single AI-discovered pattern insight.
type Pattern struct {
	Type        string `json:"type"` // "warning", "positive", "info"
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// Report is the top-level insights document.
type Report struct {
	GeneratedAt string    `json:"generated_at"`
	DailyReport string    `json:"daily_report"`
	Patterns    []Pattern `json:"patterns"`
}

// pathOverride lets tests redirect the insights file.
var pathOverride string

// Path returns ~/.break-reminder-insights.json
func Path() string {
	if pathOverride != "" {
		return pathOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".break-reminder-insights.json")
}

// Load reads the insights file. Returns (nil, nil) if the file doesn't exist.
func Load() (*Report, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Save writes the report atomically, creating parent directories if needed.
func Save(r *Report) error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/insights/ -v`
Expected: All 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/insights/
git commit -m "feat(insights): add Report/Pattern types with Save/Load"
```

---

### Task 2: Add prompt builder for AI analysis

**Files:**
- Create: `internal/insights/prompt.go`
- Create: `internal/insights/prompt_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/insights/prompt_test.go`:

```go
package insights

import (
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/ai"
)

func TestBuildPromptIncludesHistory(t *testing.T) {
	history := []ai.DailySummary{
		{Date: "2026-04-17", WorkMin: 280, BreakMin: 60, Sessions: 7, Activities: 3},
		{Date: "2026-04-16", WorkMin: 200, BreakMin: 40, Sessions: 4, Activities: 2},
	}

	prompt := BuildPrompt(history)
	if !strings.Contains(prompt, "2026-04-17") {
		t.Error("prompt missing today's date")
	}
	if !strings.Contains(prompt, "daily_report") {
		t.Error("prompt should request daily_report field")
	}
	if !strings.Contains(prompt, "patterns") {
		t.Error("prompt should request patterns field")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("prompt should request JSON format")
	}
}

func TestBuildPromptEmptyHistory(t *testing.T) {
	prompt := BuildPrompt(nil)
	if prompt == "" {
		t.Error("empty prompt for no history")
	}
	if !strings.Contains(prompt, "[]") {
		t.Error("should embed empty history array")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/insights/ -run TestBuildPrompt -v`
Expected: FAIL — BuildPrompt not defined

- [ ] **Step 3: Create prompt.go**

Create `internal/insights/prompt.go`:

```go
package insights

import (
	"encoding/json"
	"fmt"

	"github.com/devlikebear/break-reminder/internal/ai"
)

const promptTemplate = `다음은 사용자의 최근 작업/휴식 기록입니다:

%s

다음 두 가지를 분석하여 JSON으로만 응답하세요 (마크다운 코드 블록 없이):

1. daily_report: 오늘의 작업/휴식 요약을 한국어 2-3문장으로 작성
2. patterns: 눈에 띄는 패턴 2-3가지를 배열로 작성 (각 항목은 type, title, description, suggestion 필드 포함)

type 값은 다음 중 하나:
- "warning": 주의가 필요한 패턴 (예: 슬럼프, 휴식 부족)
- "positive": 긍정적 개선 추세
- "info": 중립적 관찰 (예: 최적 작업 시간대)

응답 형식:
{
  "daily_report": "...",
  "patterns": [
    {"type": "warning", "title": "...", "description": "...", "suggestion": "..."},
    {"type": "positive", "title": "...", "description": "...", "suggestion": "..."}
  ]
}`

// BuildPrompt constructs the AI prompt from the given history entries.
// Caller should trim history to desired range (e.g., last 7 days).
func BuildPrompt(history []ai.DailySummary) string {
	if history == nil {
		history = []ai.DailySummary{}
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		data = []byte("[]")
	}
	return fmt.Sprintf(promptTemplate, string(data))
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/insights/ -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/insights/prompt.go internal/insights/prompt_test.go
git commit -m "feat(insights): add prompt builder for AI analysis"
```

---

### Task 3: Add response parser

**Files:**
- Create: `internal/insights/parser.go`
- Create: `internal/insights/parser_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/insights/parser_test.go`:

```go
package insights

import (
	"strings"
	"testing"
)

func TestParseResponseValid(t *testing.T) {
	response := `{
      "daily_report": "오늘 4시간 작업했어요.",
      "patterns": [
        {"type": "warning", "title": "슬럼프", "description": "D1", "suggestion": "S1"},
        {"type": "info", "title": "골든타임", "description": "D2", "suggestion": "S2"}
      ]
    }`

	report, err := ParseResponse(response)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if report.DailyReport != "오늘 4시간 작업했어요." {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
	if len(report.Patterns) != 2 {
		t.Errorf("Patterns count = %d, want 2", len(report.Patterns))
	}
	if report.Patterns[0].Type != "warning" {
		t.Errorf("Patterns[0].Type = %q, want warning", report.Patterns[0].Type)
	}
}

func TestParseResponseStripsCodeFences(t *testing.T) {
	response := "```json\n" + `{"daily_report":"ok","patterns":[]}` + "\n```"
	report, err := ParseResponse(response)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if report.DailyReport != "ok" {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
}

func TestParseResponseInvalidJSON(t *testing.T) {
	_, err := ParseResponse("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/insights/ -run TestParseResponse -v`
Expected: FAIL — ParseResponse not defined

- [ ] **Step 3: Create parser.go**

Create `internal/insights/parser.go`:

```go
package insights

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseResponse parses the AI response into a Report (without GeneratedAt).
// Strips markdown code fences if present.
func ParseResponse(raw string) (*Report, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var r Report
	if err := json.Unmarshal([]byte(cleaned), &r); err != nil {
		return nil, fmt.Errorf("parse AI response: %w", err)
	}
	return &r, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/insights/ -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/insights/parser.go internal/insights/parser_test.go
git commit -m "feat(insights): add AI response parser"
```

---

### Task 4: Add Generate orchestration function

**Files:**
- Create: `internal/insights/generate.go`
- Create: `internal/insights/generate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/insights/generate_test.go`:

```go
package insights

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/ai"
)

type fakeAIClient struct {
	response string
	err      error
}

func (f *fakeAIClient) Query(ctx context.Context, prompt string) (string, error) {
	return f.response, f.err
}

func TestGenerateSuccess(t *testing.T) {
	client := &fakeAIClient{
		response: `{"daily_report":"test report","patterns":[{"type":"info","title":"T","description":"D","suggestion":"S"}]}`,
	}
	history := []ai.DailySummary{{Date: "2026-04-17", WorkMin: 60}}

	report, err := Generate(context.Background(), client, history, time.Now())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if report.DailyReport != "test report" {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
	if report.GeneratedAt == "" {
		t.Error("GeneratedAt should be set")
	}
}

func TestGenerateAIError(t *testing.T) {
	client := &fakeAIClient{err: errors.New("CLI not found")}
	_, err := Generate(context.Background(), client, nil, time.Now())
	if err == nil {
		t.Error("expected error from failed AI call")
	}
}

func TestGenerateInvalidResponse(t *testing.T) {
	client := &fakeAIClient{response: "not json"}
	_, err := Generate(context.Background(), client, nil, time.Now())
	if err == nil {
		t.Error("expected error from invalid AI response")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/insights/ -run TestGenerate -v`
Expected: FAIL — Generate not defined

- [ ] **Step 3: Create generate.go**

Create `internal/insights/generate.go`:

```go
package insights

import (
	"context"
	"time"

	"github.com/devlikebear/break-reminder/internal/ai"
)

// AIClient is the minimal interface Generate needs.
// The real ai.Client satisfies this automatically.
type AIClient interface {
	Query(ctx context.Context, prompt string) (string, error)
}

// Generate orchestrates building the prompt, calling the AI, and assembling the Report.
// It does NOT save the result — caller should call Save() if persistence is desired.
func Generate(ctx context.Context, client AIClient, history []ai.DailySummary, now time.Time) (*Report, error) {
	prompt := BuildPrompt(history)
	raw, err := client.Query(ctx, prompt)
	if err != nil {
		return nil, err
	}
	report, err := ParseResponse(raw)
	if err != nil {
		return nil, err
	}
	report.GeneratedAt = now.Format(time.RFC3339)
	return report, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/insights/ -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/insights/generate.go internal/insights/generate_test.go
git commit -m "feat(insights): add Generate orchestration"
```

---

### Task 5: Add `insights` CLI subcommand

**Files:**
- Create: `cmd/break-reminder/insights.go`
- Modify: `cmd/break-reminder/main.go` (register the command)

- [ ] **Step 1: Find where commands are registered**

Run: Grep to locate the root command assembly (look for `AddCommand` calls).

Look for patterns like `rootCmd.AddCommand(newDashboardCmd())` in `main.go` or nearby files.

- [ ] **Step 2: Create insights.go**

Create `cmd/break-reminder/insights.go`:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/insights"
)

func newInsightsCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Show or refresh AI insights",
		RunE: func(cmd *cobra.Command, args []string) error {
			if refresh {
				return refreshInsights()
			}
			return showInsights()
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force regenerate insights via AI CLI")
	return cmd
}

func refreshInsights() error {
	if !cfg.AIEnabled {
		return fmt.Errorf("AI is disabled in config (set ai_enabled: true)")
	}

	client := ai.NewClient(cfg.AICLI)
	if !client.Available() {
		return fmt.Errorf("AI CLI %q not found in PATH", cfg.AICLI)
	}

	history, err := ai.LoadHistory()
	if err != nil {
		return fmt.Errorf("load history: %w", err)
	}

	recent := trimRecentHistory(history, 7)

	log.Info().Int("entries", len(recent)).Msg("Generating AI insights")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	report, err := insights.Generate(ctx, client, recent, time.Now())
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if err := insights.Save(report); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	fmt.Println("Insights refreshed:")
	fmt.Println(report.DailyReport)
	return nil
}

func showInsights() error {
	report, err := insights.Load()
	if err != nil {
		return err
	}
	if report == nil {
		fmt.Println("No insights yet. Run with --refresh to generate.")
		return nil
	}
	fmt.Printf("Generated: %s\n\n", report.GeneratedAt)
	fmt.Println(report.DailyReport)
	fmt.Println()
	for _, p := range report.Patterns {
		fmt.Printf("[%s] %s\n  %s\n  → %s\n\n", p.Type, p.Title, p.Description, p.Suggestion)
	}
	return nil
}

func trimRecentHistory(history []ai.DailySummary, days int) []ai.DailySummary {
	if len(history) <= days {
		return history
	}
	return history[len(history)-days:]
}
```

- [ ] **Step 3: Register the command**

Find the location in `cmd/break-reminder/main.go` (or wherever commands are added — likely a function that returns the root `*cobra.Command`). Add to the command registration:

```go
root.AddCommand(newInsightsCmd())
```

- [ ] **Step 4: Build and verify**

Run: `go build ./cmd/break-reminder && ./break-reminder insights --help`
Expected: Help text for the `insights` command appears

- [ ] **Step 5: Commit**

```bash
git add cmd/break-reminder/insights.go cmd/break-reminder/main.go
git commit -m "feat(cli): add insights subcommand with --refresh"
```

---

### Task 6: Auto-trigger insight generation on day-end

**Files:**
- Modify: whichever file handles `ActionSaveDailyHistory` (same file touched in Phase 2 Task 4)

- [ ] **Step 1: Add auto-trigger logic**

In the same handler that receives `ActionSaveDailyHistory`, after saving the daily summary, trigger insights generation asynchronously:

```go
// After saving DailySummary history
if cfg.AIEnabled {
    go func() {
        client := ai.NewClient(cfg.AICLI)
        if !client.Available() {
            log.Warn().Str("cli", cfg.AICLI).Msg("AI CLI unavailable, skipping insights")
            return
        }
        history, err := ai.LoadHistory()
        if err != nil {
            log.Warn().Err(err).Msg("Load history for insights")
            return
        }
        recent := history
        if len(recent) > 7 {
            recent = recent[len(recent)-7:]
        }

        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
        defer cancel()

        report, err := insights.Generate(ctx, client, recent, time.Now())
        if err != nil {
            log.Warn().Err(err).Msg("Generate insights")
            return
        }
        if err := insights.Save(report); err != nil {
            log.Warn().Err(err).Msg("Save insights")
            return
        }
        log.Info().Msg("Insights auto-generated")
    }()
}
```

Add the imports: `"context"`, `"time"`, `"github.com/devlikebear/break-reminder/internal/insights"`.

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Builds cleanly

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat(service): auto-generate insights on daily history save"
```

---

### Task 7: Add InsightsLoader to HelperCore (Swift)

**Files:**
- Create: `helpers/Sources/HelperCore/InsightsLoader.swift`
- Create: `helpers/Tests/HelperCoreTests/InsightsLoaderTests.swift`

- [ ] **Step 1: Write the failing test**

Create `helpers/Tests/HelperCoreTests/InsightsLoaderTests.swift`:

```swift
import XCTest
@testable import HelperCore

final class InsightsLoaderTests: XCTestCase {
    func testParseValidReport() {
        let json = """
        {
          "generated_at": "2026-04-17T17:30:00+09:00",
          "daily_report": "오늘 요약",
          "patterns": [
            {"type":"warning","title":"T1","description":"D1","suggestion":"S1"},
            {"type":"info","title":"T2","description":"D2","suggestion":"S2"}
          ]
        }
        """
        let report = parseInsights(from: json)
        XCTAssertNotNil(report)
        XCTAssertEqual(report?.dailyReport, "오늘 요약")
        XCTAssertEqual(report?.patterns.count, 2)
        XCTAssertEqual(report?.patterns[0].type, "warning")
    }

    func testParseEmptyPatterns() {
        let json = """
        {"generated_at":"x","daily_report":"r","patterns":[]}
        """
        let report = parseInsights(from: json)
        XCTAssertNotNil(report)
        XCTAssertEqual(report?.patterns.count, 0)
    }

    func testParseInvalidReturnsNil() {
        XCTAssertNil(parseInsights(from: "not json"))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd helpers && swift test --filter InsightsLoaderTests 2>&1 | tail -10`
Expected: FAIL — parseInsights not defined

- [ ] **Step 3: Create InsightsLoader.swift**

Create `helpers/Sources/HelperCore/InsightsLoader.swift`:

```swift
import Foundation

public struct InsightPattern: Equatable {
    public let type: String       // "warning", "positive", "info"
    public let title: String
    public let description: String
    public let suggestion: String

    public init(type: String, title: String, description: String, suggestion: String) {
        self.type = type
        self.title = title
        self.description = description
        self.suggestion = suggestion
    }
}

public struct InsightsReport: Equatable {
    public let generatedAt: String
    public let dailyReport: String
    public let patterns: [InsightPattern]

    public init(generatedAt: String, dailyReport: String, patterns: [InsightPattern]) {
        self.generatedAt = generatedAt
        self.dailyReport = dailyReport
        self.patterns = patterns
    }
}

public func parseInsights(from json: String) -> InsightsReport? {
    guard let data = json.data(using: .utf8),
          let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
        return nil
    }
    let patternsRaw = dict["patterns"] as? [[String: Any]] ?? []
    let patterns = patternsRaw.map { p in
        InsightPattern(
            type: p["type"] as? String ?? "info",
            title: p["title"] as? String ?? "",
            description: p["description"] as? String ?? "",
            suggestion: p["suggestion"] as? String ?? ""
        )
    }
    return InsightsReport(
        generatedAt: dict["generated_at"] as? String ?? "",
        dailyReport: dict["daily_report"] as? String ?? "",
        patterns: patterns
    )
}

public func loadInsightsFromDisk() -> InsightsReport? {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-insights.json")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return nil }
    return parseInsights(from: content)
}
```

- [ ] **Step 4: Run tests**

Run: `cd helpers && swift test --filter InsightsLoaderTests`
Expected: All 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/HelperCore/InsightsLoader.swift helpers/Tests/HelperCoreTests/InsightsLoaderTests.swift
git commit -m "feat(helpercore): add InsightsLoader for parsing insights JSON"
```

---

### Task 8: Wire insights into DashboardViewModel

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardViewModel.swift`

- [ ] **Step 1: Add insights state and loading**

In `DashboardViewModel`, add:

```swift
@Published var insights: InsightsReport?
@Published var isRefreshingInsights = false

func loadInsights() {
    insights = loadInsightsFromDisk()
}

func refreshInsights() {
    guard !isRefreshingInsights else { return }
    isRefreshingInsights = true

    Task.detached { [weak self] in
        await self?.runInsightsRefresh()
    }
}

@MainActor
private func runInsightsRefresh() async {
    defer { isRefreshingInsights = false }

    guard let cli = findHelper("break-reminder") else {
        return
    }

    let process = Process()
    process.launchPath = cli
    process.arguments = ["insights", "--refresh"]
    process.standardOutput = FileHandle.nullDevice
    process.standardError = FileHandle.nullDevice

    do {
        try process.run()
        process.waitUntilExit()
    } catch {
        return
    }

    loadInsights()
}
```

In `start()`, after `loadHistory()`, also call `loadInsights()`. In `refresh()` (the 1-second tick), also call `loadInsights()` so the file update from the auto-trigger propagates.

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardViewModel.swift
git commit -m "feat(dashboard): wire insights loading and refresh into ViewModel"
```

---

### Task 9: Build real InsightsTabView

**Files:**
- Modify: `helpers/Sources/DashboardApp/InsightsTabView.swift`

- [ ] **Step 1: Replace placeholder with real view**

Overwrite `helpers/Sources/DashboardApp/InsightsTabView.swift`:

```swift
import SwiftUI
import AppKit
import HelperCore

struct InsightsTabView: View {
    @ObservedObject var vm: DashboardViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if let report = vm.insights {
                    dailyReportCard(report)
                    Divider().background(Color(white: 0.2))
                    patternsSection(report)
                    Divider().background(Color(white: 0.2))
                    actionButtons(report)
                } else {
                    emptyState
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 12)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "sparkles")
                .font(.system(size: 40))
                .foregroundColor(.gray)
            Text("아직 인사이트가 없습니다")
                .font(.system(size: 13))
                .foregroundColor(Color(white: 0.9))
            Text("AI CLI(claude 또는 codex)가 설치되어 있다면\n아래 버튼을 눌러 생성하세요.")
                .font(.system(size: 11))
                .foregroundColor(.gray)
                .multilineTextAlignment(.center)
            Button(action: { vm.refreshInsights() }) {
                if vm.isRefreshingInsights {
                    ProgressView().scaleEffect(0.7)
                } else {
                    Text("AI 분석 생성")
                }
            }
            .buttonStyle(DashboardButtonStyle())
            .frame(width: 140)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(.top, 40)
    }

    private func dailyReportCard(_ report: InsightsReport) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("✨ 오늘의 리포트")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(Color(white: 0.9))
                Spacer()
                Text(shortTime(report.generatedAt))
                    .font(.system(size: 10))
                    .foregroundColor(.gray)
            }

            HStack {
                Rectangle()
                    .fill(Color(red: 0.3, green: 0.8, blue: 0.5))
                    .frame(width: 3)
                Text(report.dailyReport)
                    .font(.system(size: 12))
                    .foregroundColor(Color(white: 0.85))
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.leading, 8)
            }
            .padding(10)
            .background(Color(white: 0.15))
            .cornerRadius(10)
        }
    }

    private func patternsSection(_ report: InsightsReport) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("🔍 패턴 인사이트")
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(Color(white: 0.9))

            ForEach(Array(report.patterns.enumerated()), id: \.offset) { _, p in
                patternCard(p)
            }
        }
    }

    private func patternCard(_ pattern: InsightPattern) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Circle()
                    .fill(patternColor(pattern.type))
                    .frame(width: 6, height: 6)
                Text(pattern.title)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundColor(Color(white: 0.9))
            }
            Text(pattern.description)
                .font(.system(size: 11))
                .foregroundColor(Color(white: 0.7))
                .fixedSize(horizontal: false, vertical: true)
            if !pattern.suggestion.isEmpty {
                Text("→ \(pattern.suggestion)")
                    .font(.system(size: 11))
                    .foregroundColor(Color(white: 0.6))
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(white: 0.15))
        .cornerRadius(8)
    }

    private func actionButtons(_ report: InsightsReport) -> some View {
        HStack(spacing: 12) {
            Button(action: { vm.refreshInsights() }) {
                HStack(spacing: 4) {
                    if vm.isRefreshingInsights {
                        ProgressView().scaleEffect(0.6)
                    } else {
                        Text("🔄")
                    }
                    Text("새로고침")
                }
            }
            .buttonStyle(DashboardButtonStyle())

            Button(action: { copyReport(report) }) {
                Text("📋 리포트 복사")
            }
            .buttonStyle(DashboardButtonStyle())
        }
    }

    private func patternColor(_ type: String) -> Color {
        switch type {
        case "warning": return Color(red: 1.0, green: 0.8, blue: 0.4)
        case "positive": return Color(red: 0.3, green: 0.8, blue: 0.5)
        default: return Color(red: 0.4, green: 0.7, blue: 1.0)
        }
    }

    private func shortTime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return iso }
        let display = DateFormatter()
        display.dateFormat = "HH:mm"
        return "\(display.string(from: date)) 생성"
    }

    private func copyReport(_ report: InsightsReport) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        pasteboard.setString(report.dailyReport, forType: .string)
    }
}
```

- [ ] **Step 2: Update call site to pass the ViewModel**

In `DashboardAppMain.swift`, update the switch:

```swift
case .insights:
    InsightsTabView(vm: vm)
```

- [ ] **Step 3: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add helpers/Sources/DashboardApp/InsightsTabView.swift helpers/Sources/DashboardApp/DashboardAppMain.swift
git commit -m "feat(dashboard): build real InsightsTabView with report and patterns"
```

---

### Task 10: Manual integration test with mock insights

- [ ] **Step 1: Build everything**

Run: `make build`
Expected: Success

- [ ] **Step 2: Seed mock insights for testing**

```bash
cat > ~/.break-reminder-insights.json << 'EOF'
{
  "generated_at": "2026-04-17T17:30:00+09:00",
  "daily_report": "오늘 4시간 20분 작업하고 50분 휴식했어요. 오전에 집중력이 높았고, 오후 3시 이후 휴식 간격이 짧아졌어요.",
  "patterns": [
    {"type":"warning","title":"오후 슬럼프 패턴 감지","description":"최근 5일 중 4일, 오후 2시~4시에 평균 작업 시간이 35% 줄었어요.","suggestion":"이 시간대에 짧은 산책을 추가하면 효과적일 수 있어요."},
    {"type":"positive","title":"휴식 습관 개선 중","description":"지난주 대비 휴식 건너뛰기가 40% 줄었어요.","suggestion":"꾸준히 유지하면 집중력 향상에 도움이 됩니다."},
    {"type":"info","title":"최적 작업 시간대","description":"오전 10시~12시가 가장 집중도가 높은 골든 타임이에요.","suggestion":"중요한 작업은 이 시간에 배치하면 좋겠어요."}
  ]
}
EOF
```

- [ ] **Step 3: Launch dashboard and verify Insights tab**

Run: `bin/break-dashboard`

Click the **인사이트** tab. Verify:
- Daily report card shows with green left border
- 3 pattern cards render with correct type colors (warning=yellow dot, positive=green dot, info=blue dot)
- Refresh and Copy buttons appear

Click **📋 리포트 복사**, then paste elsewhere — the daily report text should be in the clipboard.

- [ ] **Step 4: Test empty state**

Delete the insights file: `rm ~/.break-reminder-insights.json`

Relaunch dashboard, click Insights tab. Verify:
- "아직 인사이트가 없습니다" empty state appears with sparkles icon
- "AI 분석 생성" button is present

- [ ] **Step 5: Test refresh button (requires ai_enabled + installed CLI)**

If `ai_enabled: true` in config and claude/codex CLI is installed, clicking the refresh button should run the CLI and repopulate the insights file. This may take 30s-2min.

If AI isn't available, the button will silently fail — no crash — and the empty state remains. That's expected.

- [ ] **Step 6: Commit any fixes**

```bash
git add -A
git commit -m "fix(dashboard): address Phase 3 integration issues" || true
```

---

## Phase 3 Completion Checklist

- [ ] `go test ./...` passes (including new insights package tests)
- [ ] `cd helpers && swift test` passes (including InsightsLoaderTests)
- [ ] `make build` succeeds
- [ ] `break-reminder insights --refresh` generates insights when AI CLI available
- [ ] `break-reminder insights` prints current insights
- [ ] Dashboard Insights tab renders daily report and patterns correctly
- [ ] Pattern type colors (warning/positive/info) display correctly
- [ ] Refresh button triggers CLI regeneration
- [ ] Copy button copies daily report to clipboard
- [ ] Empty state shows when no insights file exists
- [ ] Day-end triggers auto-regeneration (verified by log entries)
