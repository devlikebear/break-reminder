import AppKit
import Foundation
import HelperCore

// MARK: - File I/O (platform-specific, not tested via Swift tests)

func loadStateFromFile() -> AppState {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppState() }
    return parseState(from: content)
}

func loadConfigFromFile() -> AppConfig {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".config/break-reminder/config.yaml")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppConfig() }
    return parseConfig(from: content)
}

// MARK: - Helper discovery

private func trustedHelperCandidates(named name: String) -> [String] {
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

    return candidates
}

/// Returns the path of a helper binary from trusted install locations only.
func findHelper(_ name: String) -> String? {
    for candidate in trustedHelperCandidates(named: name) {
        if FileManager.default.isExecutableFile(atPath: candidate) {
            return candidate
        }
    }
    return nil
}

/// Asks the installed CLI for its version. The helper carries no version of its
/// own, so this is the only source of truth. Returns `unknown` when the CLI is
/// missing or fails to answer.
func queryInstalledVersion() -> String {
    guard let cli = findHelper("break-reminder") else { return AboutInfo.unknownVersion }

    let process = Process()
    process.executableURL = URL(fileURLWithPath: cli)
    process.arguments = ["version"]
    let pipe = Pipe()
    process.standardOutput = pipe
    process.standardError = FileHandle.nullDevice

    do {
        try process.run()
    } catch {
        return AboutInfo.unknownVersion
    }
    let data = pipe.fileHandleForReading.readDataToEndOfFile()
    process.waitUntilExit()

    guard process.terminationStatus == 0,
          let output = String(data: data, encoding: .utf8) else {
        return AboutInfo.unknownVersion
    }
    return parseVersionOutput(output)
}

// MARK: - MenuBarController

class MenuBarController: NSObject {
    private var statusItem: NSStatusItem!
    private var refreshTimer: Timer?
    private var animationTimer: Timer?
    private var animationTick = 0
    private var currentState = AppState()
    private var currentConfig = AppConfig()

    /// Resolved on first use and kept for the lifetime of the app — the
    /// installed version cannot change while this process is running.
    private lazy var installedVersion: String = queryInstalledVersion()

    // Keep strong refs to menu items that need live updates.
    private var statusMenuItem: NSMenuItem!
    private var statsMenuItem: NSMenuItem!
    private var sessionMenuItem: NSMenuItem!

    override init() {
        super.init()
        setupStatusItem()
        statusItem.button?.imagePosition = .imageLeading
        statusItem.button?.imageScaling = .scaleProportionallyDown
        refresh()
        refreshTimer = scheduleMenuBarTimer(interval: 1.0) { [weak self] in
            self?.refresh()
        }
        animationTimer = scheduleMenuBarTimer(interval: 0.25) { [weak self] in
            self?.advanceAnimation()
        }
    }

    deinit {
        refreshTimer?.invalidate()
        animationTimer?.invalidate()
    }

    // MARK: Setup

    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

        let menu = NSMenu()

        // 1. Current status line — updated every tick, never enabled
        statusMenuItem = NSMenuItem(title: "Loading…", action: nil, keyEquivalent: "")
        statusMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)

        statsMenuItem = NSMenuItem(title: "Loading stats…", action: nil, keyEquivalent: "")
        statsMenuItem.isEnabled = false
        menu.addItem(statsMenuItem)

        menu.addItem(.separator())

        // 2. Open Dashboard
        let dashItem = NSMenuItem(title: "Open Dashboard", action: #selector(openDashboard), keyEquivalent: "d")
        dashItem.target = self
        menu.addItem(dashItem)

        // 3. Start / Stop Work Session
        sessionMenuItem = NSMenuItem(title: "Start Work Session", action: #selector(toggleSession), keyEquivalent: "s")
        sessionMenuItem.target = self
        menu.addItem(sessionMenuItem)

        // 4. Reset Timer
        let resetItem = NSMenuItem(title: "Reset Timer", action: #selector(resetTimer), keyEquivalent: "r")
        resetItem.target = self
        menu.addItem(resetItem)

        // 5. Force Break
        let breakItem = NSMenuItem(title: "Force Break", action: #selector(forceBreak), keyEquivalent: "b")
        breakItem.target = self
        menu.addItem(breakItem)

        // 6. Open Config
        let configItem = NSMenuItem(title: "Open Config", action: #selector(openConfig), keyEquivalent: ",")
        configItem.target = self
        menu.addItem(configItem)

        // 7. About
        let aboutItem = NSMenuItem(title: "About \(AboutInfo.appName)", action: #selector(showAbout), keyEquivalent: "")
        aboutItem.target = self
        menu.addItem(aboutItem)

        menu.addItem(.separator())

        // 8. Quit
        let quitItem = NSMenuItem(title: "Quit Break Reminder", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    // MARK: Refresh

    @objc private func refresh() {
        let state = loadStateFromFile()
        currentState = state
        let config = loadConfigFromFile()
        currentConfig = config
        let now = Int64(Date().timeIntervalSince1970)

        let presentation = menuBarPresentation(state: state, config: config, now: now)
        statusItem.button?.title = presentation.title
        updateMascotImage()
        statusMenuItem.title = presentation.statusLine
        statsMenuItem.title = presentation.statsLine
        sessionMenuItem.title = state.isSessionActive ? "Stop Work Session" : "Start Work Session"
    }

    private func advanceAnimation() {
        animationTick = (animationTick + 1) % 12_000
        updateMascotImage()
    }

    private func updateMascotImage() {
        let frame = menuBarAnimation(state: currentState, config: currentConfig, tick: animationTick)
        statusItem.button?.image = HamsterMenuBarIcon.image(for: frame)
    }

    // MARK: Actions

    @objc private func openDashboard() {
        guard let helperPath = findHelper("break-dashboard") else {
            showAlert(message: "break-dashboard helper not found.",
                      info: "Run 'make build' or 'make install' so helpers are placed next to the break-reminder binary.")
            return
        }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: helperPath)
        try? task.run()
    }

    @objc private func resetTimer() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let path = home.appendingPathComponent(".break-reminder-state")
        let priorState = loadStateFromFile()
        let config = loadConfigFromFile()
        let now = Int64(Date().timeIntervalSince1970)
        let totals = liveDailyTotals(state: priorState, config: config, now: now)
        var s = AppState()
        s.lastCheck = now
        s.todayWorkSeconds = totals.workSeconds
        s.todayBreakSeconds = totals.breakSeconds
        s.lastUpdateDate = totals.date
        s.todayPomodoros = priorState.todayPomodoros
        carryOverSession(from: priorState, into: &s)
        try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
        refresh()
    }

    @objc private func forceBreak() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let path = home.appendingPathComponent(".break-reminder-state")
        let state = loadStateFromFile()
        let config = loadConfigFromFile()
        let now = Int64(Date().timeIntervalSince1970)
        let totals = liveDailyTotals(state: state, config: config, now: now)
        var s = AppState()
        s.mode = "break"
        s.lastCheck = now
        s.breakStart = now
        s.todayWorkSeconds = totals.workSeconds
        s.todayBreakSeconds = totals.breakSeconds
        s.lastUpdateDate = totals.date
        s.pomodoroCount = state.pomodoroCount
        s.todayPomodoros = state.todayPomodoros
        carryOverSession(from: state, into: &s)
        try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
        refresh()
    }

    /// Menu-bar writes rebuild the state file from scratch, so the work-session
    /// fields owned by the Go timer have to be carried over explicitly.
    private func carryOverSession(from previous: AppState, into next: inout AppState) {
        next.sessionState = previous.sessionState
        next.sessionStart = previous.sessionStart
        next.sessionEnd = previous.sessionEnd
        next.sessionManual = previous.sessionManual
        next.lastActivity = previous.lastActivity
    }

    @objc private func toggleSession() {
        guard let helperPath = findHelper("break-reminder") else {
            showAlert(message: "break-reminder binary not found.",
                      info: "Run 'make install' so the CLI is placed next to the menu bar helper.")
            return
        }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: helperPath)
        task.arguments = [currentState.isSessionActive ? "stop" : "start"]
        try? task.run()
        task.waitUntilExit()
        refresh()
    }

    @objc private func showAbout() {
        // Accessory apps are not frontmost, so the panel would open behind
        // whatever the user is working in unless we activate first.
        NSApp.activate(ignoringOtherApps: true)

        let alert = NSAlert()
        alert.messageText = AboutInfo.appName
        alert.informativeText = "Version \(installedVersion)"
        alert.alertStyle = .informational
        alert.addButton(withTitle: "GitHub")
        alert.addButton(withTitle: "Close")

        if alert.runModal() == .alertFirstButtonReturn,
           let url = URL(string: AboutInfo.repositoryURL) {
            NSWorkspace.shared.open(url)
        }
    }

    @objc private func openConfig() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let configPath = home
            .appendingPathComponent(".config/break-reminder/config.yaml")
        NSWorkspace.shared.open(configPath)
    }

    // MARK: Helpers

    private func showAlert(message: String, info: String) {
        let alert = NSAlert()
        alert.messageText = message
        alert.informativeText = info
        alert.alertStyle = .warning
        alert.runModal()
    }
}

// MARK: - Entry point

let app = NSApplication.shared
app.setActivationPolicy(.accessory)   // menu-bar-only; no Dock icon

let controller = MenuBarController()

// Keep controller alive for the lifetime of the app.
withExtendedLifetime(controller) {
    app.run()
}
