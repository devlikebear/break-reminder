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
    private var pomodoroItem: NSMenuItem!
    private var classicItem: NSMenuItem!
    private var pomodoroStatusItem: NSMenuItem!
    private var pomodoroBusy = false
    private let toolsFiles = TimeToolsFiles()
    private var toolsSnapshot = TimeToolsSnapshot()
    private var toolsBusy = false
    private var toolsMenuItem: NSMenuItem!
    private var toolsStatusItem: NSMenuItem?
    private var toolsMenuKey = ""


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
        menu.autoenablesItems = false

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

        menu.addItem(.separator())
        pomodoroStatusItem = NSMenuItem(title: "", action: nil, keyEquivalent: "")
        menu.addItem(pomodoroStatusItem)
        pomodoroItem = NSMenuItem(title: L10n.text("Start Pomodoro"), action: #selector(startPomodoro), keyEquivalent: "")
        pomodoroItem.target = self; menu.addItem(pomodoroItem)
        classicItem = NSMenuItem(title: L10n.text("Switch to classic cycle"), action: #selector(selectClassic), keyEquivalent: "")
        classicItem.target = self; menu.addItem(classicItem)

        toolsMenuItem = NSMenuItem(title: L10n.text("Countdown"), action: nil, keyEquivalent: "")
        menu.addItem(toolsMenuItem)

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
        let quitItem = NSMenuItem(title: L10n.text("Quit menu bar (timers keep running)"), action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
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
        pomodoroStatusItem.title = pomodoroStatus(state: state, config: config) + L10n.text(" · activity-based")
        pomodoroItem.isEnabled = !pomodoroBusy && !pomodoroStartCommands(state: state, config: config).isEmpty
        classicItem.isEnabled = !pomodoroBusy && config.pomodoroEnabled && !state.paused && state.mode == "work"
        pomodoroItem.state = config.pomodoroEnabled ? .on : .off
        sessionMenuItem.title = state.isSessionActive ? "Stop Work Session" : "Start Work Session"
        refreshTimeTools(nowMS: now * 1000)
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

    private func refreshTimeTools(nowMS: Int64) {
        let runtime = toolsFiles.loadRuntime()
        let running = runtime.isRunning(nowMS: nowMS)
        var readError: String?
        do { toolsSnapshot = try toolsFiles.loadSnapshot() } catch { readError = error.localizedDescription }
        if readError == nil, let title = timeToolsMenuTitle(snapshot: toolsSnapshot, nowMS: nowMS) { statusItem.button?.title = title }
        let key = "\(toolsSnapshot.revision):\(running):\(toolsBusy):\(runtime.notificationAvailable):\(readError ?? "")"
        if key != toolsMenuKey {
            toolsMenuKey = key
            let menu = NSMenu(); menu.autoenablesItems = false
            let status = NSMenuItem(title: "", action: nil, keyEquivalent: ""); status.isEnabled = false
            toolsStatusItem = status; menu.addItem(status)
            if let readError {
                let errorItem = NSMenuItem(title: L10n.text("Read failed: ") + readError, action: nil, keyEquivalent: "")
                errorItem.isEnabled = false; menu.addItem(errorItem)
            }
            if !running {
                let setup = NSMenuItem(title: L10n.text("Set up / repair time tools…"), action: #selector(setupTimeTools), keyEquivalent: "")
                setup.target = self; setup.isEnabled = !toolsBusy; menu.addItem(setup)
            } else if !runtime.notificationAvailable {
                let warning = NSMenuItem(title: L10n.text("Notifications unavailable · check completion in the app"), action: nil, keyEquivalent: "")
                warning.isEnabled = false; menu.addItem(warning)
            }
            if let c = toolsSnapshot.countdown {
                if c.isActive {
                    menu.addItem(toolItem(c.phase == "running" ? L10n.text("Pause") : L10n.text("Resume"), command: ["timer", c.phase == "running" ? "pause" : "resume", "--id", c.id], enabled: c.phase == "running" || running))
                    menu.addItem(toolItem(L10n.text("Cancel timer…"), command: ["timer", "cancel", "--id", c.id]))
                } else {
                    menu.addItem(toolItem(L10n.text("Restart"), command: ["timer", "restart", "--id", c.id], enabled: running))
                }
            }
            for event in toolsSnapshot.unread.reversed() {
                menu.addItem(toolItem(L10n.text("Acknowledge: ") + String(event.displayLabel.prefix(32)), command: ["acknowledge", "--event-id", event.id]))
                if event.deliveryState == "failed" || event.deliveryState == "unknown" {
                    menu.addItem(toolItem(L10n.text("Retry notification"), command: ["notify-again", "--event-id", event.id], enabled: running))
                }
            }
            menu.addItem(.separator())
            for minutes in [5, 10, 15, 30] {
                var args = ["timer", "start", "--duration", "\(minutes)m"]
                if let c = toolsSnapshot.countdown, c.isActive { args += ["--replace-id", c.id] }
                menu.addItem(toolItem(L10n.text("Start {0} min", minutes), command: args, enabled: running))
            }
            let custom = NSMenuItem(title: L10n.text("Custom timer…"), action: #selector(openTimerDashboard), keyEquivalent: "")
            custom.target = self; menu.addItem(custom)
            if readError != nil { for item in menu.items where item.action == #selector(timeToolsAction(_:)) { item.isEnabled = false } }
            toolsMenuItem.submenu = menu
        }
        if let c = toolsSnapshot.countdown {
            toolsStatusItem?.title = String(c.displayLabel.prefix(32)) + " · " + c.timeText(nowMS: nowMS) + (c.phase == "paused" ? L10n.text(" · paused") : c.phase == "completed" ? L10n.text(" · completed") : "")
        } else { toolsStatusItem?.title = running ? L10n.text("No active timer") : L10n.text("Background timers are stopped") }
    }

    private func toolItem(_ title: String, command: [String], enabled: Bool = true) -> NSMenuItem {
        let item = NSMenuItem(title: title, action: #selector(timeToolsAction(_:)), keyEquivalent: "")
        item.target = self; item.isEnabled = enabled && !toolsBusy
        item.representedObject = command + ["--if-revision", String(toolsSnapshot.revision)]
        return item
    }

    @objc private func timeToolsAction(_ sender: NSMenuItem) {
        guard let command = sender.representedObject as? [String], !toolsBusy else { return }
        if command.contains("--replace-id") || command.prefix(2) == ["timer", "cancel"] {
            let alert = NSAlert(); alert.messageText = L10n.text("Cancel the current timer?")
            alert.informativeText = command.contains("--replace-id") ? L10n.text("Cancel this timer and start a new one.") : L10n.text("Cancel without a completion notification.")
            alert.addButton(withTitle: L10n.text("Continue")); alert.addButton(withTitle: L10n.text("Go back"))
            guard alert.runModal() == .alertFirstButtonReturn else { return }
        }
        runTimeToolsCLI(toolsFiles.arguments(command))
    }

    @objc private func setupTimeTools() {
        let alert = NSAlert(); alert.messageText = L10n.text("Set up the background time-tools service?")
        alert.informativeText = L10n.text("Install and start work reminders, the menu bar, and timers.")
        alert.addButton(withTitle: L10n.text("Set up")); alert.addButton(withTitle: L10n.text("Cancel"))
        if alert.runModal() == .alertFirstButtonReturn { runTimeToolsCLI(["service", "install"]) }
    }

    private func runTimeToolsCLI(_ args: [String]) {
        guard !toolsBusy else { return }
        toolsBusy = true; refresh()
        Task { @MainActor in
            let result = await CLICommandClient(executablePath: findHelper("break-reminder")).run(arguments: args)
            self.toolsBusy = false; self.refresh()
            if !result.succeeded { self.showAlert(message: L10n.text("Time-tools command failed"), info: result.errorMessage) }
        }
    }

    @objc private func openTimerDashboard() {
        guard let path = findHelper("break-dashboard") else { return }
        let task = Process(); task.executableURL = URL(fileURLWithPath: path); task.arguments = ["--time-tools"]
        do { try task.run() } catch { showAlert(message: L10n.text("Could not open dashboard"), info: error.localizedDescription) }
    }

    @objc private func startPomodoro() {
        runPomodoro(pomodoroStartCommands(state: currentState, config: currentConfig))
    }

    @objc private func selectClassic() { runPomodoro([["pomodoro", "off"]]) }

    private func runPomodoro(_ commands: [[String]]) {
        guard !pomodoroBusy, !commands.isEmpty else { return }
        if currentState.isSessionActive {
            let alert = NSAlert()
            alert.messageText = L10n.text("Reset the current work interval and switch modes?")
            alert.informativeText = L10n.text("Today's work totals will be kept.")
            alert.addButton(withTitle: L10n.text("Switch")); alert.addButton(withTitle: L10n.text("Cancel"))
            guard alert.runModal() == .alertFirstButtonReturn else { return }
        }
        pomodoroBusy = true; refresh()
        Task { @MainActor in
            let actions = PomodoroActions()
            await actions.perform(commands: commands, client: CLICommandClient(executablePath: findHelper("break-reminder")))
            self.pomodoroBusy = false; self.refresh()
            if let message = actions.message { self.showAlert(message: L10n.text("Pomodoro command failed"), info: message) }
        }
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
