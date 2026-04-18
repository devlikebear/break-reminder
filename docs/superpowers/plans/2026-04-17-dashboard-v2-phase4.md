# DashboardApp v2 Phase 4: Visual Enhancements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add mascot with state-aware messaging, SwiftUI animations (ring glow, state transitions, confetti, tab slide), and a theme system (dark/light/auto) to make the dashboard feel alive.

**Architecture:** `MascotEngine` (in HelperCore) maps state+config+conditions to `(emoji, message)`. A `ThemeManager` `ObservableObject` holds the current color palette, swappable via the new `theme` config field. Animations use native SwiftUI primitives (`withAnimation`, `.transition`, `Canvas + TimelineView`).

**Tech Stack:** SwiftUI, HelperCore, Go (config schema extension)

**Prerequisites:** Phase 3 complete (all tabs, insights)

---

### Task 1: Add theme field to Go Config

**Files:**
- Modify: `internal/config/types.go:4-27`
- Modify: `internal/config/defaults.go` (find it via grep if needed)
- Modify: `internal/config/load.go` (merge function)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go` (append at end):

```go
func TestThemeDefaultAuto(t *testing.T) {
	cfg := Default()
	if cfg.Theme != "auto" {
		t.Errorf("Theme default = %q, want 'auto'", cfg.Theme)
	}
}

func TestThemeOverrideFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := configDir
	defer func() { configDir = origDir }()

	home, _ := os.UserHomeDir()
	rel, _ := filepath.Rel(home, tmpDir)
	configDir = rel

	path := filepath.Join(tmpDir, configFile)
	content := `theme: dark
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want 'dark'", cfg.Theme)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestTheme -v`
Expected: FAIL — Theme field doesn't exist

- [ ] **Step 3: Add Theme field**

Edit `internal/config/types.go` — add a field to the `Config` struct:

```go
Theme string `yaml:"theme"` // "auto", "dark", "light"
```

Find `Default()` in the config package (grep for `func Default`) and add:

```go
Theme: "auto",
```

Find the `merge()` function in `load.go` and add:

```go
if src.Theme != "" {
    dst.Theme = src.Theme
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/ -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add theme field (auto/dark/light)"
```

---

### Task 2: Add theme to Swift AppConfig (HelperCore)

**Files:**
- Modify: `helpers/Sources/HelperCore/ConfigParser.swift`
- Modify: `helpers/Tests/HelperCoreTests/ConfigParserTests.swift`

- [ ] **Step 1: Write the failing test**

Add to `helpers/Tests/HelperCoreTests/ConfigParserTests.swift` (append at end):

```swift
func testThemeDefaultAuto() {
    let cfg = AppConfig()
    XCTAssertEqual(cfg.theme, "auto")
}

func testThemeParseFromYAML() {
    let yaml = """
    work_duration_min: 50
    theme: dark
    """
    let cfg = parseConfig(from: yaml)
    XCTAssertEqual(cfg.theme, "dark")
}

func testThemeParseLight() {
    let cfg = parseConfig(from: "theme: light")
    XCTAssertEqual(cfg.theme, "light")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd helpers && swift test --filter ConfigParserTests 2>&1 | tail -10`
Expected: FAIL — theme property doesn't exist

- [ ] **Step 3: Update ConfigParser**

Edit `helpers/Sources/HelperCore/ConfigParser.swift`:

```swift
public struct AppConfig: Equatable {
    public var workDurationMin: Int = 50
    public var breakDurationMin: Int = 10
    public var idleThresholdSec: Int = 120
    public var checkIntervalSec: Int = 60
    public var theme: String = "auto"

    public init() {}
}

public func parseConfig(from content: String) -> AppConfig {
    var c = AppConfig()
    for line in content.components(separatedBy: "\n") {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        let parts = trimmed.split(separator: ":", maxSplits: 1)
        guard parts.count == 2 else { continue }
        let key = String(parts[0]).trimmingCharacters(in: .whitespaces)
        let val = String(parts[1]).trimmingCharacters(in: .whitespaces)
        switch key {
        case "work_duration_min":  c.workDurationMin = Int(val) ?? 50
        case "break_duration_min": c.breakDurationMin = Int(val) ?? 10
        case "idle_threshold_sec": c.idleThresholdSec = Int(val) ?? 120
        case "check_interval_sec": c.checkIntervalSec = Int(val) ?? 60
        case "theme":              c.theme = val
        default: break
        }
    }
    return c
}
```

- [ ] **Step 4: Run tests**

Run: `cd helpers && swift test --filter ConfigParserTests`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/HelperCore/ConfigParser.swift helpers/Tests/HelperCoreTests/ConfigParserTests.swift
git commit -m "feat(helpercore): add theme field to AppConfig"
```

---

### Task 3: Create ThemeManager in DashboardApp

**Files:**
- Create: `helpers/Sources/DashboardApp/ThemeManager.swift`

- [ ] **Step 1: Create ThemeManager**

Create `helpers/Sources/DashboardApp/ThemeManager.swift`:

```swift
import SwiftUI

enum ThemeMode: String {
    case auto, dark, light

    init(raw: String) {
        self = ThemeMode(rawValue: raw) ?? .auto
    }
}

@MainActor
final class ThemeManager: ObservableObject {
    @Published var mode: ThemeMode = .auto
    @Published var systemIsDark: Bool = true // Updated from Environment

    var isDark: Bool {
        switch mode {
        case .dark: return true
        case .light: return false
        case .auto: return systemIsDark
        }
    }

    // MARK: - Color tokens

    var background: Color {
        isDark ? Color(red: 0.102, green: 0.102, blue: 0.118) : Color(red: 0.961, green: 0.961, blue: 0.969)
    }

    var surface: Color {
        isDark ? Color(red: 0.145, green: 0.145, blue: 0.157) : Color.white
    }

    var textPrimary: Color {
        isDark ? Color(white: 0.9) : Color(red: 0.102, green: 0.102, blue: 0.118)
    }

    var textSecondary: Color {
        isDark ? Color(white: 0.5) : Color(white: 0.4)
    }

    var accent: Color {
        isDark ? Color(red: 0.302, green: 0.800, blue: 0.502) : Color(red: 0.204, green: 0.659, blue: 0.325)
    }

    var accentBreak: Color {
        isDark ? Color(red: 0.400, green: 0.702, blue: 1.000) : Color(red: 0.259, green: 0.522, blue: 0.957)
    }

    var warning: Color {
        isDark ? Color(red: 1.0, green: 0.8, blue: 0.4) : Color(red: 0.976, green: 0.671, blue: 0.000)
    }

    var divider: Color {
        Color(white: isDark ? 0.2 : 0.85)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/ThemeManager.swift
git commit -m "feat(dashboard): add ThemeManager with color tokens"
```

---

### Task 4: Inject ThemeManager and sync with config

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`
- Modify: `helpers/Sources/DashboardApp/DashboardViewModel.swift`

- [ ] **Step 1: Wire ThemeManager into the app**

In `DashboardAppMain.swift`, add `@StateObject` for the theme manager and inject it into the view tree:

```swift
@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()
    @StateObject private var theme = ThemeManager()
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .environmentObject(theme)
                .frame(width: 360, height: 600)
                .background(theme.background)
                .onAppear {
                    vm.start()
                    theme.mode = ThemeMode(raw: vm.config.theme)
                    configureWindow()
                }
                .onDisappear { vm.stop() }
                .onKeyPress("q") { NSApp.terminate(nil); return .handled }
                .onKeyPress("r") { vm.resetTimer(); return .handled }
                .onKeyPress("b") { vm.forceBreak(); return .handled }
                .onChange(of: vm.config.theme) { _, newValue in
                    theme.mode = ThemeMode(raw: newValue)
                }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
        .defaultPosition(.topTrailing)
    }

    // (configureWindow unchanged)
}
```

In `DashboardContentView`, pick up theme from environment and use it to drive colors. Also detect system color scheme:

```swift
struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager
    @Environment(\.colorScheme) private var systemColorScheme

    private var accentColor: Color {
        if vm.isPaused { return theme.warning }
        return vm.isWork ? theme.accent : theme.accentBreak
    }

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(theme.divider)
            TabBarView(selectedTab: $vm.selectedTab, accentColor: accentColor)

            Group {
                switch vm.selectedTab {
                case .timer:
                    TimerTabView(vm: vm)
                case .stats:
                    StatsTabView(vm: vm)
                case .insights:
                    InsightsTabView(vm: vm)
                }
            }
        }
        .onChange(of: systemColorScheme) { _, newValue in
            theme.systemIsDark = (newValue == .dark)
        }
        .onAppear {
            theme.systemIsDark = (systemColorScheme == .dark)
        }
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardAppMain.swift
git commit -m "feat(dashboard): inject ThemeManager and sync with config"
```

---

### Task 5: Update views to use ThemeManager colors

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
- Modify: `helpers/Sources/DashboardApp/TimerTabView.swift`
- Modify: `helpers/Sources/DashboardApp/StatsTabView.swift`
- Modify: `helpers/Sources/DashboardApp/InsightsTabView.swift`
- Modify: `helpers/Sources/DashboardApp/TabBarView.swift`

- [ ] **Step 1: Update StatusHeaderView**

Replace hardcoded colors in `StatusHeaderView.swift`:

```swift
struct StatusHeaderView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager

    private var statusColor: Color {
        if vm.isPaused { return theme.warning }
        return vm.isWork ? theme.accent : theme.accentBreak
    }

    private var ringSize: CGFloat { 140 }

    var body: some View {
        VStack(spacing: 12) {
            HStack {
                Circle()
                    .fill(statusColor)
                    .frame(width: 10, height: 10)
                Text(vm.statusText)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundColor(statusColor)
                Spacer()
                Text(vm.modeDetail)
                    .font(.system(size: 12))
                    .foregroundColor(theme.textSecondary)
            }

            ZStack {
                CircularProgressRing(
                    progress: vm.sessionProgress.progress,
                    fillColor: statusColor,
                    trackColor: theme.divider,
                    lineWidth: 10
                )
                .frame(width: ringSize, height: ringSize)

                VStack(spacing: 2) {
                    Text(vm.sessionProgress.remainingFormatted)
                        .font(.system(size: 32, weight: .ultraLight).monospacedDigit())
                        .foregroundColor(theme.textPrimary)
                    Text(vm.sessionSubtitle)
                        .font(.system(size: 11))
                        .foregroundColor(theme.textSecondary)
                }
            }
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
    }
}
```

- [ ] **Step 2: Update CircularProgressRing to accept track color**

Edit `helpers/Sources/DashboardApp/CircularProgressRing.swift`:

```swift
import SwiftUI

struct CircularProgressRing: View {
    let progress: Double
    let fillColor: Color
    let trackColor: Color
    let lineWidth: CGFloat

    init(progress: Double, fillColor: Color, trackColor: Color = Color(white: 0.2), lineWidth: CGFloat = 10) {
        self.progress = progress
        self.fillColor = fillColor
        self.trackColor = trackColor
        self.lineWidth = lineWidth
    }

    var body: some View {
        ZStack {
            Circle()
                .stroke(trackColor, lineWidth: lineWidth)

            Circle()
                .trim(from: 0, to: CGFloat(min(progress, 1.0)))
                .stroke(fillColor, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                .rotationEffect(.degrees(-90))
        }
    }
}
```

- [ ] **Step 3: Update TimerTabView**

Replace colors in `TimerTabView.swift` — use `@EnvironmentObject var theme: ThemeManager` and replace:

- `Color(red: 0.3, green: 0.8, blue: 0.5)` → `theme.accent`
- `Color(red: 0.4, green: 0.7, blue: 1.0)` → `theme.accentBreak`
- `Color(white: 0.9)` → `theme.textPrimary`
- `.gray` for secondary text → `theme.textSecondary`
- `Color(white: 0.2)` for track → `theme.divider`
- Add `@EnvironmentObject var theme: ThemeManager` to `TimerTabView` struct

- [ ] **Step 4: Update StatsTabView, InsightsTabView, TabBarView**

Apply the same pattern: add `@EnvironmentObject var theme: ThemeManager` and swap hardcoded colors for theme tokens.

For `DashboardButtonStyle`, since it's a ButtonStyle struct (not a View), it can't directly use EnvironmentObject. Add a theme parameter:

```swift
struct DashboardButtonStyle: ButtonStyle {
    let surfaceColor: Color
    let textColor: Color

    init(surfaceColor: Color = Color(white: 0.22), textColor: Color = Color(white: 0.9)) {
        self.surfaceColor = surfaceColor
        self.textColor = textColor
    }

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .medium))
            .foregroundColor(textColor)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(surfaceColor.opacity(configuration.isPressed ? 1.3 : 1.0))
            )
    }
}
```

Call sites update to: `.buttonStyle(DashboardButtonStyle(surfaceColor: theme.surface, textColor: theme.textPrimary))`

- [ ] **Step 5: Verify build and run**

Run: `cd helpers && swift build -c release && .build/release/DashboardApp`
Expected:
- Dashboard launches in current system theme
- Change system preferences (System Settings → Appearance) to light mode — dashboard should switch colors
- Set `theme: light` in `~/.config/break-reminder/config.yaml` — dashboard should stay light regardless of system theme

- [ ] **Step 6: Commit**

```bash
git add helpers/Sources/DashboardApp/
git commit -m "feat(dashboard): use ThemeManager colors across all views"
```

---

### Task 6: Create MascotEngine in HelperCore

**Files:**
- Create: `helpers/Sources/HelperCore/MascotEngine.swift`
- Create: `helpers/Tests/HelperCoreTests/MascotEngineTests.swift`

- [ ] **Step 1: Write the failing test**

Create `helpers/Tests/HelperCoreTests/MascotEngineTests.swift`:

```swift
import XCTest
@testable import HelperCore

final class MascotEngineTests: XCTestCase {
    func testWorkingStateReturnsHamster() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 300 // 5 min
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "🐹")
        XCTAssertFalse(mascot.message.isEmpty)
    }

    func testBreakStateReturnsSleeping() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = Int64(Date().timeIntervalSince1970) - 60
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😴")
    }

    func testLongWorkReturnsConcerned() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 7200 // 2 hours
        let config = AppConfig()
        config.workDurationMin = 50 // break every 50 min
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😰")
    }

    func testPausedReturnsNeutral() {
        var state = AppState()
        state.mode = "work"
        state.paused = true
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😶")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd helpers && swift test --filter MascotEngineTests 2>&1 | tail -10`
Expected: FAIL — mascotFor not defined

- [ ] **Step 3: Create MascotEngine.swift**

Create `helpers/Sources/HelperCore/MascotEngine.swift`:

```swift
import Foundation

public struct Mascot: Equatable {
    public let emoji: String
    public let message: String

    public init(emoji: String, message: String) {
        self.emoji = emoji
        self.message = message
    }
}

/// Selects a mascot (emoji + message) based on the current state.
public func mascotFor(state: AppState, config: AppConfig, now: Int64) -> Mascot {
    // Paused
    if state.paused {
        return Mascot(emoji: "😶", message: "일시 정지 중이에요")
    }

    // Break mode
    if state.mode == "break" {
        let breakElapsed = state.breakStart > 0 ? Int(now - state.breakStart) : 0
        let breakTotal = config.breakDurationMin * 60
        if breakElapsed > breakTotal - 60 {
            return Mascot(emoji: "☕", message: "곧 다시 시작해요~")
        }
        return Mascot(emoji: "😴", message: "푹 쉬고 와요~ ☕")
    }

    // Work mode
    let workTotal = config.workDurationMin * 60
    let elapsed = state.workSeconds

    // Long continuous work warning (2x the configured work duration)
    if elapsed >= workTotal * 2 {
        return Mascot(emoji: "😰", message: "쉬어가는 게 어때요? 🙏")
    }

    // Near break time (last 5 minutes)
    if workTotal - elapsed <= 300 && workTotal - elapsed > 0 {
        return Mascot(emoji: "🐹", message: "곧 휴식 시간이에요~ ☕")
    }

    // Default working
    return Mascot(emoji: "🐹", message: "집중 모드! 화이팅 💪")
}

/// Selects a mascot for achievement moments (e.g., daily goal).
public func mascotForAchievement(dailyWorkMinutes: Int, goalMinutes: Int) -> Mascot? {
    guard goalMinutes > 0, dailyWorkMinutes >= goalMinutes else { return nil }
    return Mascot(emoji: "🎉", message: "오늘도 잘 해냈어요! 🏆")
}
```

- [ ] **Step 4: Run tests**

Run: `cd helpers && swift test --filter MascotEngineTests`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/HelperCore/MascotEngine.swift helpers/Tests/HelperCoreTests/MascotEngineTests.swift
git commit -m "feat(helpercore): add MascotEngine with state-based emoji/message mapping"
```

---

### Task 7: Add mascot to StatusHeaderView

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
- Modify: `helpers/Sources/DashboardApp/DashboardViewModel.swift`

- [ ] **Step 1: Compute mascot in ViewModel**

Add to `DashboardViewModel.swift`:

```swift
var currentMascot: Mascot {
    mascotFor(state: state, config: config, now: now)
}
```

- [ ] **Step 2: Render mascot in StatusHeaderView**

Edit `helpers/Sources/DashboardApp/StatusHeaderView.swift` — add a mascot row at the bottom of the VStack:

```swift
var body: some View {
    VStack(spacing: 12) {
        // ... existing status row ...
        // ... existing ZStack with ring ...

        mascotRow
    }
    .padding(.horizontal, 20)
    .padding(.top, 16)
    .padding(.bottom, 12)
}

private var mascotRow: some View {
    HStack(spacing: 8) {
        Text(vm.currentMascot.emoji)
            .font(.system(size: 22))
            .id(vm.currentMascot.emoji) // Triggers transition on change

        Text(vm.currentMascot.message)
            .font(.system(size: 11))
            .foregroundColor(theme.textSecondary)
            .lineLimit(2)
    }
    .padding(.horizontal, 12)
    .padding(.vertical, 6)
    .background(
        RoundedRectangle(cornerRadius: 12)
            .fill(theme.surface)
    )
    .frame(maxWidth: .infinity)
}
```

- [ ] **Step 3: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 4: Launch and verify**

Run: `.build/release/DashboardApp`
Expected: Mascot row appears below the timer ring with 🐹 and a message

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/DashboardApp/
git commit -m "feat(dashboard): add mascot row to status header"
```

---

### Task 8: Add state-transition animation (work ↔ break color morph)

**Files:**
- Modify: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`

- [ ] **Step 1: Animate mode changes**

In `StatusHeaderView.swift`, wrap the color-dependent views in `.animation()`:

```swift
var body: some View {
    VStack(spacing: 12) {
        // status row ...
        // ring ZStack ...
        mascotRow
    }
    .padding(.horizontal, 20)
    .padding(.top, 16)
    .padding(.bottom, 12)
    .animation(.easeInOut(duration: 0.5), value: vm.isWork)
    .animation(.easeInOut(duration: 0.3), value: vm.isPaused)
}
```

Also add mascot bounce:

```swift
private var mascotRow: some View {
    HStack(spacing: 8) {
        Text(vm.currentMascot.emoji)
            .font(.system(size: 22))
            .scaleEffect(vm.isPaused ? 0.9 : 1.0)
            .animation(.spring(response: 0.4, dampingFraction: 0.6), value: vm.currentMascot.emoji)
        // ...
    }
    // ...
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Launch and verify**

Run: `.build/release/DashboardApp`

Click **Force Break** — observe the color morph from green to blue over 0.5s. Click **Reset** — color morphs back.

- [ ] **Step 4: Commit**

```bash
git add helpers/Sources/DashboardApp/StatusHeaderView.swift
git commit -m "feat(dashboard): add color morph animation on state transitions"
```

---

### Task 9: Add tab transition animation

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`

- [ ] **Step 1: Add transition to Group**

Update `DashboardContentView.body`:

```swift
Group {
    switch vm.selectedTab {
    case .timer:
        TimerTabView(vm: vm)
            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
    case .stats:
        StatsTabView(vm: vm)
            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
    case .insights:
        InsightsTabView(vm: vm)
            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
    }
}
.animation(.easeInOut(duration: 0.25), value: vm.selectedTab)
```

- [ ] **Step 2: Verify build and test**

Run: `.build/release/DashboardApp`

Click different tabs — content should slide in/out instead of hard-cutting.

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardAppMain.swift
git commit -m "feat(dashboard): add slide transition between tabs"
```

---

### Task 10: Add progress ring glow effect

**Files:**
- Modify: `helpers/Sources/DashboardApp/CircularProgressRing.swift`
- Modify: `helpers/Sources/DashboardApp/StatusHeaderView.swift`

- [ ] **Step 1: Add glow effect**

Edit `CircularProgressRing.swift`:

```swift
import SwiftUI

struct CircularProgressRing: View {
    let progress: Double
    let fillColor: Color
    let trackColor: Color
    let lineWidth: CGFloat

    init(progress: Double, fillColor: Color, trackColor: Color = Color(white: 0.2), lineWidth: CGFloat = 10) {
        self.progress = progress
        self.fillColor = fillColor
        self.trackColor = trackColor
        self.lineWidth = lineWidth
    }

    var body: some View {
        ZStack {
            Circle()
                .stroke(trackColor, lineWidth: lineWidth)

            Circle()
                .trim(from: 0, to: CGFloat(min(progress, 1.0)))
                .stroke(fillColor, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                .rotationEffect(.degrees(-90))
                .shadow(color: fillColor.opacity(0.6), radius: 4)
                .animation(.easeInOut(duration: 1.0), value: progress)
        }
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Visual check**

Run: `.build/release/DashboardApp`

The progress ring should have a subtle glow around the filled portion. As the progress increases (once per second), the fill animates smoothly.

- [ ] **Step 4: Commit**

```bash
git add helpers/Sources/DashboardApp/CircularProgressRing.swift
git commit -m "feat(dashboard): add glow effect to progress ring"
```

---

### Task 11: Add confetti for daily goal achievement

**Files:**
- Create: `helpers/Sources/DashboardApp/ConfettiView.swift`
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`
- Modify: `helpers/Sources/DashboardApp/DashboardViewModel.swift`

- [ ] **Step 1: Create ConfettiView using Canvas + TimelineView**

Create `helpers/Sources/DashboardApp/ConfettiView.swift`:

```swift
import SwiftUI

struct ConfettiParticle: Identifiable {
    let id = UUID()
    let x: CGFloat         // Start x position (0...1)
    let delay: TimeInterval
    let duration: TimeInterval
    let color: Color
    let rotationSpeed: Double

    static func random(colors: [Color]) -> ConfettiParticle {
        ConfettiParticle(
            x: .random(in: 0...1),
            delay: .random(in: 0...0.5),
            duration: .random(in: 1.5...3.0),
            color: colors.randomElement() ?? .green,
            rotationSpeed: .random(in: 0.5...2.0)
        )
    }
}

struct ConfettiView: View {
    let particles: [ConfettiParticle]
    let isActive: Bool

    static func generate(count: Int, colors: [Color]) -> [ConfettiParticle] {
        (0..<count).map { _ in ConfettiParticle.random(colors: colors) }
    }

    var body: some View {
        if isActive {
            TimelineView(.animation) { timeline in
                Canvas { context, size in
                    let elapsed = timeline.date.timeIntervalSinceReferenceDate
                        .truncatingRemainder(dividingBy: 3.5)

                    for particle in particles {
                        let t = max(0, elapsed - particle.delay)
                        guard t < particle.duration else { continue }
                        let progress = t / particle.duration

                        let x = particle.x * size.width + sin(t * 3) * 20
                        let y = progress * (size.height + 40)
                        let rotation = t * particle.rotationSpeed * .pi * 2

                        var transform = CGAffineTransform(translationX: x, y: y)
                        transform = transform.rotated(by: rotation)

                        let rect = CGRect(x: -4, y: -6, width: 8, height: 12).applying(transform)
                        context.fill(
                            Path(rect),
                            with: .color(particle.color.opacity(1.0 - progress))
                        )
                    }
                }
            }
            .allowsHitTesting(false)
        }
    }
}
```

- [ ] **Step 2: Add goal tracking to ViewModel**

Add to `DashboardViewModel.swift`:

```swift
@Published var showConfetti = false
private var lastGoalCheckMinute = 0
private let dailyGoalMinutes = 240 // 4 hours default — could come from config

func checkGoalAchievement() {
    let workMin = dailyTotals.workSeconds / 60
    if workMin >= dailyGoalMinutes && lastGoalCheckMinute < dailyGoalMinutes {
        showConfetti = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 3.5) { [weak self] in
            self?.showConfetti = false
        }
    }
    lastGoalCheckMinute = workMin
}
```

Call `checkGoalAchievement()` at the end of `refresh()`.

- [ ] **Step 3: Overlay confetti on content view**

In `DashboardAppMain.swift`:

```swift
struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager
    @Environment(\.colorScheme) private var systemColorScheme

    @State private var confettiParticles: [ConfettiParticle] = []

    private var accentColor: Color {
        if vm.isPaused { return theme.warning }
        return vm.isWork ? theme.accent : theme.accentBreak
    }

    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                StatusHeaderView(vm: vm)
                Divider().background(theme.divider)
                TabBarView(selectedTab: $vm.selectedTab, accentColor: accentColor)

                Group {
                    switch vm.selectedTab {
                    case .timer:
                        TimerTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    case .stats:
                        StatsTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    case .insights:
                        InsightsTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    }
                }
                .animation(.easeInOut(duration: 0.25), value: vm.selectedTab)
            }

            ConfettiView(
                particles: confettiParticles,
                isActive: vm.showConfetti
            )
            .onAppear {
                confettiParticles = ConfettiView.generate(
                    count: 50,
                    colors: [theme.accent, theme.accentBreak, theme.warning, .pink, .purple]
                )
            }
        }
        .onChange(of: systemColorScheme) { _, newValue in
            theme.systemIsDark = (newValue == .dark)
        }
        .onAppear {
            theme.systemIsDark = (systemColorScheme == .dark)
        }
    }
}
```

- [ ] **Step 4: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 5: Test confetti manually**

To test, temporarily lower the goal threshold in the ViewModel (e.g., `dailyGoalMinutes = 1`) and reach 1 minute of work. Or seed the state file with `TODAY_WORK_SECONDS=14400` (4 hours × 60 × 60) and launch — confetti should trigger on the first refresh.

Restore `dailyGoalMinutes = 240` before committing.

- [ ] **Step 6: Commit**

```bash
git add helpers/Sources/DashboardApp/ConfettiView.swift helpers/Sources/DashboardApp/DashboardAppMain.swift helpers/Sources/DashboardApp/DashboardViewModel.swift
git commit -m "feat(dashboard): add confetti animation on daily goal achievement"
```

---

### Task 12: Full Phase 4 integration verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... && cd helpers && swift test`
Expected: All tests pass

- [ ] **Step 2: Run build**

Run: `make build`
Expected: Success

- [ ] **Step 3: Launch and verify**

Run: `bin/break-dashboard`

Verify:
- Mascot row renders below timer ring
- Force break → colors morph smoothly
- Reset → colors morph back
- Switching tabs → slide animation
- Ring has subtle glow
- Theme config works:
  - `theme: auto` → follows system
  - `theme: dark` → always dark
  - `theme: light` → always light (switch system to light mode to confirm; dashboard stays light on `dark` setting)

Test from CLI:
Run: `bin/break-reminder dashboard --gui`
Expected: Same dashboard launches

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "fix(dashboard): address Phase 4 integration issues" || true
```

---

## Phase 4 Completion Checklist

- [ ] `go test ./...` passes
- [ ] `cd helpers && swift test` passes (including MascotEngineTests)
- [ ] `make build` succeeds
- [ ] Mascot row appears below timer ring with state-appropriate emoji/message
- [ ] Color morph animates on work↔break transitions
- [ ] Tab transitions slide smoothly
- [ ] Progress ring has glow effect
- [ ] Theme system works: auto/dark/light all render correctly
- [ ] `theme: light` in config overrides system theme
- [ ] Confetti triggers when daily goal is reached (manual verification)
- [ ] Mascot shows concerned face when working 2x work duration
- [ ] Break mode mascot shows sleeping face with rest message
