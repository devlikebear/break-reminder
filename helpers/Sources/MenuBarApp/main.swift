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

/// Returns the path of a helper binary by searching next to the running
/// executable, then bin/, then ~/.local/bin/, then PATH.
func findHelper(_ name: String) -> String? {
    if let exe = Bundle.main.executablePath {
        let candidate = URL(fileURLWithPath: exe)
            .deletingLastPathComponent()
            .appendingPathComponent(name).path
        if FileManager.default.fileExists(atPath: candidate) { return candidate }
    }

    let home = FileManager.default.homeDirectoryForCurrentUser.path
    let candidates = [
        "bin/\(name)",
        "\(home)/.local/bin/\(name)",
    ]
    for c in candidates {
        if FileManager.default.fileExists(atPath: c) { return c }
    }

    // Fall back to PATH lookup
    let task = Process()
    task.launchPath = "/usr/bin/which"
    task.arguments = [name]
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = FileHandle.nullDevice
    if (try? task.run()) != nil {
        task.waitUntilExit()
        if task.terminationStatus == 0 {
            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            if let p = String(data: data, encoding: .utf8)?
                .trimmingCharacters(in: .whitespacesAndNewlines), !p.isEmpty {
                return p
            }
        }
    }
    return nil
}

// MARK: - MenuBarController

class MenuBarController: NSObject {
    private var statusItem: NSStatusItem!
    private var refreshTimer: Timer?

    // Keep strong refs to menu items that need live updates.
    private var statusMenuItem: NSMenuItem!

    override init() {
        super.init()
        setupStatusItem()
        refresh()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.refresh()
        }
    }

    // MARK: Setup

    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

        let menu = NSMenu()

        // 1. Current status line — updated every tick, never enabled
        statusMenuItem = NSMenuItem(title: "Loading…", action: nil, keyEquivalent: "")
        statusMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)

        menu.addItem(.separator())

        // 2. Open Dashboard
        let dashItem = NSMenuItem(title: "Open Dashboard", action: #selector(openDashboard), keyEquivalent: "d")
        dashItem.target = self
        menu.addItem(dashItem)

        // 3. Reset Timer
        let resetItem = NSMenuItem(title: "Reset Timer", action: #selector(resetTimer), keyEquivalent: "r")
        resetItem.target = self
        menu.addItem(resetItem)

        // 4. Force Break
        let breakItem = NSMenuItem(title: "Force Break", action: #selector(forceBreak), keyEquivalent: "b")
        breakItem.target = self
        menu.addItem(breakItem)

        // 5. Open Config
        let configItem = NSMenuItem(title: "Open Config", action: #selector(openConfig), keyEquivalent: ",")
        configItem.target = self
        menu.addItem(configItem)

        menu.addItem(.separator())

        // 6. Quit
        let quitItem = NSMenuItem(title: "Quit Break Reminder", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    // MARK: Refresh

    @objc private func refresh() {
        let state = loadStateFromFile()
        let config = loadConfigFromFile()
        let now = Int64(Date().timeIntervalSince1970)

        let isWork = state.mode == "work"

        // -- Title text --
        let title: String
        if isWork {
            let sp = workProgress(state: state, config: config, now: now)
            let elapsedMin = sp.elapsedSec / 60
            title = "Work \(elapsedMin)m/\(config.workDurationMin)m"
        } else {
            let sp = breakProgress(state: state, config: config, now: now)
            let remainMin = max(sp.remainingSec / 60, 0)
            title = "Break \(remainMin)m left"
        }

        statusItem.button?.title = title

        // -- Disabled status menu item --
        statusMenuItem.title = isWork ? "Working…" : "On Break…"
    }

    // MARK: Actions

    @objc private func openDashboard() {
        guard let helperPath = findHelper("break-dashboard") else {
            showAlert(message: "break-dashboard helper not found.",
                      info: "Run 'make build-helper' to build it.")
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
        let now = Int64(Date().timeIntervalSince1970)
        let df = DateFormatter()
        df.dateFormat = "yyyy-MM-dd"
        var s = AppState()
        s.lastCheck = now
        s.todayWorkSeconds = priorState.todayWorkSeconds
        s.todayBreakSeconds = priorState.todayBreakSeconds
        s.lastUpdateDate = df.string(from: Date())
        try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
        refresh()
    }

    @objc private func forceBreak() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let path = home.appendingPathComponent(".break-reminder-state")
        let state = loadStateFromFile()
        let now = Int64(Date().timeIntervalSince1970)
        var s = AppState()
        s.mode = "break"
        s.lastCheck = now
        s.breakStart = now
        s.todayWorkSeconds = state.todayWorkSeconds
        s.todayBreakSeconds = state.todayBreakSeconds
        s.lastUpdateDate = state.lastUpdateDate
        try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
        refresh()
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
