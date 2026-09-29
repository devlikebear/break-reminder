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
        pomodoroItem = NSMenuItem(title: "포모도로 시작", action: #selector(startPomodoro), keyEquivalent: "")
        pomodoroItem.target = self; menu.addItem(pomodoroItem)
        classicItem = NSMenuItem(title: "기본 주기로 전환", action: #selector(selectClassic), keyEquivalent: "")
        classicItem.target = self; menu.addItem(classicItem)

        toolsMenuItem = NSMenuItem(title: "일반 타이머", action: nil, keyEquivalent: "")
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
        let quitItem = NSMenuItem(title: "메뉴바 종료 (타이머는 계속 실행)", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
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
        pomodoroStatusItem.title = pomodoroStatus(state: state, config: config) + " · 활동 감지 기준"
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
                let errorItem = NSMenuItem(title: "읽기 실패: " + readError, action: nil, keyEquivalent: "")
                errorItem.isEnabled = false; menu.addItem(errorItem)
            }
            if !running {
                let setup = NSMenuItem(title: "시간 도구 설정 / 복구…", action: #selector(setupTimeTools), keyEquivalent: "")
                setup.target = self; setup.isEnabled = !toolsBusy; menu.addItem(setup)
            } else if !runtime.notificationAvailable {
                let warning = NSMenuItem(title: "시스템 알림 없음 · 완료는 앱에서 확인", action: nil, keyEquivalent: "")
                warning.isEnabled = false; menu.addItem(warning)
            }
            if let c = toolsSnapshot.countdown {
                if c.isActive {
                    menu.addItem(toolItem(c.phase == "running" ? "일시정지" : "재개", command: ["timer", c.phase == "running" ? "pause" : "resume", "--id", c.id], enabled: c.phase == "running" || running))
                    menu.addItem(toolItem("타이머 취소…", command: ["timer", "cancel", "--id", c.id]))
                } else {
                    menu.addItem(toolItem("다시 시작", command: ["timer", "restart", "--id", c.id], enabled: running))
                }
            }
            for event in toolsSnapshot.unread.reversed() {
                menu.addItem(toolItem("완료 확인: " + String(event.label.prefix(32)), command: ["acknowledge", "--event-id", event.id]))
                if event.deliveryState == "failed" || event.deliveryState == "unknown" {
                    menu.addItem(toolItem("알림 다시 보내기", command: ["notify-again", "--event-id", event.id], enabled: running))
                }
            }
            menu.addItem(.separator())
            for minutes in [5, 10, 15, 30] {
                var args = ["timer", "start", "--duration", "\(minutes)m"]
                if let c = toolsSnapshot.countdown, c.isActive { args += ["--replace-id", c.id] }
                menu.addItem(toolItem("\(minutes)분 시작", command: args, enabled: running))
            }
            let custom = NSMenuItem(title: "직접 설정…", action: #selector(openTimerDashboard), keyEquivalent: "")
            custom.target = self; menu.addItem(custom)
            if readError != nil { for item in menu.items where item.action == #selector(timeToolsAction(_:)) { item.isEnabled = false } }
            toolsMenuItem.submenu = menu
        }
        if let c = toolsSnapshot.countdown {
            toolsStatusItem?.title = String(c.label.prefix(32)) + " · " + c.timeText(nowMS: nowMS) + (c.phase == "paused" ? " · 일시정지" : c.phase == "completed" ? " · 완료" : "")
        } else { toolsStatusItem?.title = running ? "실행 중인 타이머 없음" : "백그라운드 타이머가 중지됐습니다" }
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
            let alert = NSAlert(); alert.messageText = "현재 타이머를 취소할까요?"
            alert.informativeText = command.contains("--replace-id") ? "취소 후 새 타이머를 시작합니다." : "완료 알림 없이 취소합니다."
            alert.addButton(withTitle: "계속"); alert.addButton(withTitle: "돌아가기")
            guard alert.runModal() == .alertFirstButtonReturn else { return }
        }
        runTimeToolsCLI(toolsFiles.arguments(command))
    }

    @objc private func setupTimeTools() {
        let alert = NSAlert(); alert.messageText = "시간 도구 백그라운드 서비스를 설정할까요?"
        alert.informativeText = "업무 알림·메뉴바·타이머 서비스를 설치하고 시작합니다."
        alert.addButton(withTitle: "설정"); alert.addButton(withTitle: "취소")
        if alert.runModal() == .alertFirstButtonReturn { runTimeToolsCLI(["service", "install"]) }
    }

    private func runTimeToolsCLI(_ args: [String]) {
        guard !toolsBusy else { return }
        toolsBusy = true; refresh()
        Task { @MainActor in
            let result = await CLICommandClient(executablePath: findHelper("break-reminder")).run(arguments: args)
            self.toolsBusy = false; self.refresh()
            if !result.succeeded { self.showAlert(message: "시간 도구 명령 실패", info: result.errorMessage) }
        }
    }

    @objc private func openTimerDashboard() {
        guard let path = findHelper("break-dashboard") else { return }
        let task = Process(); task.executableURL = URL(fileURLWithPath: path); task.arguments = ["--time-tools"]
        do { try task.run() } catch { showAlert(message: "대시보드를 열지 못했습니다", info: error.localizedDescription) }
    }

    @objc private func startPomodoro() {
        runPomodoro(pomodoroStartCommands(state: currentState, config: currentConfig))
    }

    @objc private func selectClassic() { runPomodoro([["pomodoro", "off"]]) }

    private func runPomodoro(_ commands: [[String]]) {
        guard !pomodoroBusy, !commands.isEmpty else { return }
        if currentState.isSessionActive {
            let alert = NSAlert()
            alert.messageText = "현재 작업 구간을 초기화하고 모드를 전환할까요?"
            alert.informativeText = "오늘 작업 누계는 유지됩니다."
            alert.addButton(withTitle: "전환"); alert.addButton(withTitle: "취소")
            guard alert.runModal() == .alertFirstButtonReturn else { return }
        }
        pomodoroBusy = true; refresh()
        Task { @MainActor in
            let actions = PomodoroActions()
            await actions.perform(commands: commands, client: CLICommandClient(executablePath: findHelper("break-reminder")))
            self.pomodoroBusy = false; self.refresh()
            if let message = actions.message { self.showAlert(message: "포모도로 명령 실패", info: message) }
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
