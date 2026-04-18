# DashboardApp v2 Phase 1: SwiftUI Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the existing AppKit DashboardApp to SwiftUI while preserving all current functionality (status display, circular progress ring, daily stats, system info, action buttons, keyboard shortcuts).

**Architecture:** Replace `DashboardApp/main.swift` (single 429-line AppKit file) with a SwiftUI `App` entry point and composable views. HelperCore dependencies (`StateParser`, `ConfigParser`, `ProgressCalc`, `TimeFormatter`) remain unchanged. Data flow stays the same: 1-second polling of `~/.break-reminder-state` and `~/.config/break-reminder/config.yaml`.

**Tech Stack:** SwiftUI, AppKit (NSWindow configuration only), HelperCore

---

### Task 1: Update Package.swift for macOS 13 minimum

**Files:**
- Modify: `helpers/Package.swift`

- [ ] **Step 1: Update platform target**

```swift
// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "BreakReminderHelpers",
    platforms: [.macOS(.v13)],
    targets: [
        .target(
            name: "HelperCore"
        ),
        .executableTarget(
            name: "BreakScreenApp",
            dependencies: ["HelperCore"]
        ),
        .executableTarget(
            name: "DashboardApp",
            dependencies: ["HelperCore"]
        ),
        .executableTarget(
            name: "MenuBarApp",
            dependencies: ["HelperCore"]
        ),
        .testTarget(
            name: "HelperCoreTests",
            dependencies: ["HelperCore"]
        ),
    ]
)
```

- [ ] **Step 2: Verify existing tests still pass**

Run: `cd helpers && swift test`
Expected: All tests pass (platform bump doesn't break existing code)

- [ ] **Step 3: Commit**

```bash
git add helpers/Package.swift
git commit -m "chore: bump macOS deployment target to 13 for Swift Charts"
```

---

### Task 2: Create DashboardViewModel with timer polling

**Files:**
- Create: `helpers/Sources/DashboardApp/DashboardViewModel.swift`
- Test: `helpers/Tests/HelperCoreTests/DashboardViewModelTests.swift` (logic is in HelperCore, but we test the polling integration)

- [ ] **Step 1: Write the ViewModel test**

Create `helpers/Tests/HelperCoreTests/DashboardViewModelTests.swift`:

```swift
import XCTest
@testable import HelperCore

final class DashboardViewModelTests: XCTestCase {
    func testWorkProgressCalculation() {
        let state = AppState()
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let sp = workProgress(state: state, config: config, now: now)
        XCTAssertGreaterThanOrEqual(sp.progress, 0.0)
        XCTAssertLessThanOrEqual(sp.progress, 1.0)
    }

    func testBreakProgressCalculation() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = Int64(Date().timeIntervalSince1970) - 60
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let sp = breakProgress(state: state, config: config, now: now)
        XCTAssertGreaterThan(sp.elapsedSec, 0)
    }

    func testLiveDailyTotalsDefaultState() {
        let state = AppState()
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let totals = liveDailyTotals(state: state, config: config, now: now)
        XCTAssertGreaterThanOrEqual(totals.workSeconds, 0)
        XCTAssertGreaterThanOrEqual(totals.breakSeconds, 0)
    }
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd helpers && swift test --filter DashboardViewModelTests`
Expected: PASS (these test existing HelperCore functions)

- [ ] **Step 3: Create the ViewModel**

Create `helpers/Sources/DashboardApp/DashboardViewModel.swift`:

```swift
import Foundation
import SwiftUI
import HelperCore

@MainActor
final class DashboardViewModel: ObservableObject {
    @Published var state: AppState = AppState()
    @Published var config: AppConfig = AppConfig()
    @Published var idleSeconds: Int = 0
    @Published var launchdStatusText: String = "Unknown"

    private var timer: Timer?

    var isWork: Bool { state.mode == "work" }
    var isPaused: Bool { state.paused }
    var now: Int64 { Int64(Date().timeIntervalSince1970) }

    var sessionProgress: SessionProgress {
        if isWork {
            return workProgress(state: state, config: config, now: now)
        } else {
            return breakProgress(state: state, config: config, now: now)
        }
    }

    var dailyTotals: LiveDailyTotals {
        liveDailyTotals(state: state, config: config, now: now)
    }

    var statusText: String {
        if isPaused {
            return "PAUSED (\(isWork ? "WORK" : "BREAK"))"
        }
        return isWork ? "WORKING" : "ON BREAK"
    }

    var modeDetail: String {
        let sp = sessionProgress
        if isWork {
            return "\(sp.elapsedSec / 60) / \(config.workDurationMin) min"
        } else {
            return "\(sp.elapsedSec / 60) / \(config.breakDurationMin) min"
        }
    }

    var sessionSubtitle: String {
        if isPaused { return "paused" }
        return isWork ? "until break" : "until work"
    }

    func start() {
        refresh()
        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.refresh()
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }

    func refresh() {
        state = loadStateFromDisk()
        config = loadConfigFromDisk()
        idleSeconds = getIdleSecondsFromSystem()
        launchdStatusText = queryLaunchdStatus()
    }

    func resetTimer() {
        let totals = dailyTotals
        var s = AppState()
        s.lastCheck = now
        s.todayWorkSeconds = totals.workSeconds
        s.todayBreakSeconds = totals.breakSeconds
        s.lastUpdateDate = totals.date
        writeStateToDisk(s)
        refresh()
    }

    func forceBreak() {
        let totals = dailyTotals
        var s = AppState()
        s.mode = "break"
        s.lastCheck = now
        s.breakStart = now
        s.todayWorkSeconds = totals.workSeconds
        s.todayBreakSeconds = totals.breakSeconds
        s.lastUpdateDate = totals.date
        writeStateToDisk(s)
        refresh()
    }
}
```

- [ ] **Step 4: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardViewModel.swift helpers/Tests/HelperCoreTests/DashboardViewModelTests.swift
git commit -m "feat(dashboard): add DashboardViewModel with timer polling"
```

---

### Task 3: Create system I/O helper functions

These are the platform-specific functions that the ViewModel calls. They are extracted from the current `main.swift` and placed in a dedicated file.

**Files:**
- Create: `helpers/Sources/DashboardApp/SystemIO.swift`

- [ ] **Step 1: Create SystemIO.swift**

Extract the existing I/O functions from `main.swift` into `helpers/Sources/DashboardApp/SystemIO.swift`:

```swift
import Foundation
import HelperCore

func loadStateFromDisk() -> AppState {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppState() }
    return parseState(from: content)
}

func loadConfigFromDisk() -> AppConfig {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".config/break-reminder/config.yaml")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppConfig() }
    return parseConfig(from: content)
}

func writeStateToDisk(_ s: AppState) {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
}

func queryLaunchdStatus() -> String {
    let task = Process()
    task.launchPath = "/bin/launchctl"
    task.arguments = ["list", "com.devlikebear.break-reminder"]
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = pipe
    do {
        try task.run()
        task.waitUntilExit()
        return task.terminationStatus == 0 ? "Running (launchd)" : "Not loaded"
    } catch {
        return "Unknown"
    }
}

func getIdleSecondsFromSystem() -> Int {
    let task = Process()
    task.launchPath = "/usr/sbin/ioreg"
    task.arguments = ["-c", "IOHIDSystem", "-d", "4"]
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = FileHandle.nullDevice
    do {
        try task.run()
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        task.waitUntilExit()
        guard let output = String(data: data, encoding: .utf8) else { return 0 }
        for line in output.components(separatedBy: "\n") {
            if line.contains("HIDIdleTime") {
                let parts = line.components(separatedBy: "=")
                if let last = parts.last {
                    let cleaned = last.trimmingCharacters(in: .whitespacesAndNewlines)
                    if let ns = Int64(cleaned) {
                        return Int(ns / 1_000_000_000)
                    }
                }
            }
        }
    } catch {}
    return 0
}

func findHelper(_ name: String) -> String? {
    var candidates: [String] = []
    if let exe = Bundle.main.executablePath {
        candidates.append(
            URL(fileURLWithPath: exe)
                .deletingLastPathComponent()
                .appendingPathComponent(name)
                .path
        )
    }
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    candidates.append("\(home)/.local/bin/\(name)")
    for candidate in candidates {
        if FileManager.default.isExecutableFile(atPath: candidate) {
            return candidate
        }
    }
    return nil
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -5`
Expected: Build succeeds (may have warnings about unused in main.swift — that's fine, we replace it next)

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/SystemIO.swift
git commit -m "refactor(dashboard): extract system I/O functions to SystemIO.swift"
```

---

### Task 4: Create CircularProgressRing SwiftUI view

**Files:**
- Create: `helpers/Sources/DashboardApp/CircularProgressRing.swift`

- [ ] **Step 1: Create the SwiftUI progress ring**

Create `helpers/Sources/DashboardApp/CircularProgressRing.swift`:

```swift
import SwiftUI

struct CircularProgressRing: View {
    let progress: Double
    let fillColor: Color
    let lineWidth: CGFloat

    init(progress: Double, fillColor: Color, lineWidth: CGFloat = 10) {
        self.progress = progress
        self.fillColor = fillColor
        self.lineWidth = lineWidth
    }

    var body: some View {
        ZStack {
            Circle()
                .stroke(Color(white: 0.2), lineWidth: lineWidth)

            Circle()
                .trim(from: 0, to: CGFloat(min(progress, 1.0)))
                .stroke(fillColor, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                .rotationEffect(.degrees(-90))
        }
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/CircularProgressRing.swift
git commit -m "feat(dashboard): add CircularProgressRing SwiftUI view"
```

---

### Task 5: Create StatusHeaderView (fixed top section)

**Files:**
- Create: `helpers/Sources/DashboardApp/StatusHeaderView.swift`

- [ ] **Step 1: Create the status header view**

Create `helpers/Sources/DashboardApp/StatusHeaderView.swift`:

```swift
import SwiftUI
import HelperCore

struct StatusHeaderView: View {
    @ObservedObject var vm: DashboardViewModel

    private var statusColor: Color {
        if vm.isPaused { return .yellow }
        return vm.isWork ? Color(red: 0.3, green: 0.8, blue: 0.5) : Color(red: 0.4, green: 0.7, blue: 1.0)
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
                    .foregroundColor(.gray)
            }

            ZStack {
                CircularProgressRing(
                    progress: vm.sessionProgress.progress,
                    fillColor: statusColor,
                    lineWidth: 10
                )
                .frame(width: ringSize, height: ringSize)

                VStack(spacing: 2) {
                    Text(vm.sessionProgress.remainingFormatted)
                        .font(.system(size: 32, weight: .ultraLight).monospacedDigit())
                        .foregroundColor(Color(white: 0.9))
                    Text(vm.sessionSubtitle)
                        .font(.system(size: 11))
                        .foregroundColor(.gray)
                }
            }
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/StatusHeaderView.swift
git commit -m "feat(dashboard): add StatusHeaderView with progress ring"
```

---

### Task 6: Create TimerTabView (daily stats + system info + buttons)

**Files:**
- Create: `helpers/Sources/DashboardApp/TimerTabView.swift`

- [ ] **Step 1: Create the timer tab view**

Create `helpers/Sources/DashboardApp/TimerTabView.swift`:

```swift
import SwiftUI
import HelperCore

struct TimerTabView: View {
    @ObservedObject var vm: DashboardViewModel

    private let workColor = Color(red: 0.3, green: 0.8, blue: 0.5)
    private let breakColor = Color(red: 0.4, green: 0.7, blue: 1.0)

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            dailyStatsSection
            Divider().background(Color(white: 0.2))
            systemInfoSection
            Spacer()
            actionButtons
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }

    private var dailyStatsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Daily Statistics")
                .font(.system(size: 14, weight: .semibold))
                .foregroundColor(Color(white: 0.9))

            let totals = vm.dailyTotals
            let workMin = totals.workSeconds / 60
            let breakMin = totals.breakSeconds / 60
            let totalMin = workMin + breakMin

            HStack {
                Text("Work: \(formatMinutes(workMin))")
                    .font(.system(size: 13))
                    .foregroundColor(Color(white: 0.9))
                Spacer()
                Text("Break: \(formatMinutes(breakMin))")
                    .font(.system(size: 13))
                    .foregroundColor(breakColor)
            }

            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(Color(white: 0.2))
                        .frame(height: 6)
                    if totalMin > 0 {
                        RoundedRectangle(cornerRadius: 3)
                            .fill(workColor)
                            .frame(width: geo.size.width * CGFloat(workMin) / CGFloat(totalMin), height: 6)
                    }
                }
            }
            .frame(height: 6)

            if totalMin > 0 {
                HStack {
                    Spacer()
                    Text("\(workMin * 100 / totalMin)%")
                        .font(.system(size: 11))
                        .foregroundColor(.gray)
                }
            }
        }
    }

    private var systemInfoSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("System: \(vm.launchdStatusText)")
                .font(.system(size: 12))
                .foregroundColor(.gray)
            Text("Idle: \(vm.idleSeconds)s / Threshold: \(vm.config.idleThresholdSec)s")
                .font(.system(size: 12))
                .foregroundColor(.gray)
        }
    }

    private var actionButtons: some View {
        HStack(spacing: 12) {
            Button("Reset") { vm.resetTimer() }
                .buttonStyle(DashboardButtonStyle())
            Button("Force Break") { vm.forceBreak() }
                .buttonStyle(DashboardButtonStyle())
        }
    }
}

struct DashboardButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .medium))
            .foregroundColor(Color(white: 0.9))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color(white: configuration.isPressed ? 0.28 : 0.22))
            )
    }
}
```

- [ ] **Step 2: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -3`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/TimerTabView.swift
git commit -m "feat(dashboard): add TimerTabView with daily stats and action buttons"
```

---

### Task 7: Create SwiftUI App entry point and replace main.swift

**Files:**
- Create: `helpers/Sources/DashboardApp/DashboardAppMain.swift`
- Delete: `helpers/Sources/DashboardApp/main.swift`

- [ ] **Step 1: Create the new SwiftUI entry point**

Create `helpers/Sources/DashboardApp/DashboardAppMain.swift`:

```swift
import SwiftUI
import HelperCore

@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .frame(width: 360, height: 600)
                .background(Color(red: 0.1, green: 0.1, blue: 0.12))
                .onAppear { vm.start() }
                .onDisappear { vm.stop() }
                .onKeyPress("q") { NSApp.terminate(nil); return .handled }
                .onKeyPress("r") { vm.resetTimer(); return .handled }
                .onKeyPress("b") { vm.forceBreak(); return .handled }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
        .defaultPosition(.topTrailing)
    }
}

struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TimerTabView(vm: vm)
        }
    }
}
```

- [ ] **Step 2: Delete old main.swift**

Run: `rm helpers/Sources/DashboardApp/main.swift`

- [ ] **Step 3: Verify build**

Run: `cd helpers && swift build 2>&1 | tail -5`
Expected: Build succeeds. If there are issues with `@main` and the old `main.swift` conflicting, ensure the old file is deleted.

- [ ] **Step 4: Verify the app launches**

Run: `cd helpers && swift build -c release && .build/release/DashboardApp`
Expected: A 360×600 dark window appears in the top-right corner with:
- Status dot + "WORKING" label
- Circular progress ring with countdown
- Daily Statistics section
- System info
- Reset / Force Break buttons

Press `q` to quit.

- [ ] **Step 5: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardAppMain.swift helpers/Sources/DashboardApp/SystemIO.swift
git rm helpers/Sources/DashboardApp/main.swift
git commit -m "feat(dashboard): migrate DashboardApp to SwiftUI entry point"
```

---

### Task 8: Verify full build pipeline and window behavior

**Files:**
- No new files — integration verification

- [ ] **Step 1: Run all tests**

Run: `cd helpers && swift test`
Expected: All existing HelperCoreTests pass + new DashboardViewModelTests pass

- [ ] **Step 2: Run Go tests**

Run: `go test ./...`
Expected: All Go tests pass (no Go changes in Phase 1, just sanity check)

- [ ] **Step 3: Run full build via Makefile**

Run: `make build`
Expected: Both Go binary and Swift helpers build successfully. `bin/break-dashboard` exists.

- [ ] **Step 4: Test the built binary**

Run: `bin/break-dashboard`
Expected: Dashboard window appears with correct state from `~/.break-reminder-state`. Verify:
- Status updates every second
- Reset button works (resets work timer)
- Force Break button works (switches to break mode)
- `q` key quits the app
- Window is floating (stays on top)

- [ ] **Step 5: Test via Go CLI**

Run: `bin/break-reminder dashboard --gui`
Expected: Same dashboard window launches via the Go CLI wrapper

- [ ] **Step 6: Commit any fixes**

If any fixes were needed during verification:
```bash
git add -A
git commit -m "fix(dashboard): address SwiftUI migration issues"
```

---

### Task 9: Window refinements (floating, position, drag)

The SwiftUI `Window` API may not support all the AppKit window customizations we need (floating level, movable by background). We use `NSWindow` access to apply these.

**Files:**
- Modify: `helpers/Sources/DashboardApp/DashboardAppMain.swift`

- [ ] **Step 1: Add window configuration via NSApplication delegate**

Update `DashboardAppMain.swift` — add a window configurator that runs after the window appears:

```swift
import SwiftUI
import HelperCore
import AppKit

@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .frame(width: 360, height: 600)
                .background(Color(red: 0.1, green: 0.1, blue: 0.12))
                .onAppear {
                    vm.start()
                    configureWindow()
                }
                .onDisappear { vm.stop() }
                .onKeyPress("q") { NSApp.terminate(nil); return .handled }
                .onKeyPress("r") { vm.resetTimer(); return .handled }
                .onKeyPress("b") { vm.forceBreak(); return .handled }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
        .defaultPosition(.topTrailing)
    }

    private func configureWindow() {
        DispatchQueue.main.async {
            guard let window = NSApp.windows.first(where: { $0.title == "Break Reminder" }) else { return }
            window.level = .floating
            window.isMovableByWindowBackground = true
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
        }
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }
}

struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TimerTabView(vm: vm)
        }
    }
}
```

- [ ] **Step 2: Test window behavior**

Run: `cd helpers && swift build -c release && .build/release/DashboardApp`
Expected:
- Window floats above other windows
- Window is draggable by clicking anywhere on the background
- Title bar is hidden/transparent
- App quits when window is closed

- [ ] **Step 3: Commit**

```bash
git add helpers/Sources/DashboardApp/DashboardAppMain.swift
git commit -m "feat(dashboard): configure floating window with AppKit bridge"
```

---

## Phase 1 Completion Checklist

After all tasks are done, verify:

- [ ] `make build` succeeds
- [ ] `make test` succeeds (both Go and Swift)
- [ ] `bin/break-dashboard` launches and shows correct state
- [ ] `bin/break-reminder dashboard --gui` launches the same dashboard
- [ ] Status updates every second
- [ ] Reset and Force Break buttons work
- [ ] Keyboard shortcuts (q/r/b) work
- [ ] Window is floating, draggable, hidden titlebar
- [ ] Break mode shows blue colors, work mode shows green
- [ ] Paused state shows yellow
