import AppKit
import Foundation

// MARK: - State & Config Parsing

struct AppState {
    var workSeconds: Int = 0
    var mode: String = "work"
    var lastCheck: Int64 = 0
    var breakStart: Int64 = 0
    var todayWorkSeconds: Int = 0
    var todayBreakSeconds: Int = 0
    var lastUpdateDate: String = ""
}

struct AppConfig {
    var workDurationMin: Int = 50
    var breakDurationMin: Int = 10
    var idleThresholdSec: Int = 120
    var checkIntervalSec: Int = 60
}

func loadState() -> AppState {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppState() }

    var s = AppState()
    for line in content.components(separatedBy: "\n") {
        let parts = line.split(separator: "=", maxSplits: 1)
        guard parts.count == 2 else { continue }
        let key = String(parts[0])
        let val = String(parts[1])
        switch key {
        case "WORK_SECONDS":      s.workSeconds = Int(val) ?? 0
        case "MODE":              s.mode = val
        case "LAST_CHECK":        s.lastCheck = Int64(val) ?? 0
        case "BREAK_START":       s.breakStart = Int64(val) ?? 0
        case "TODAY_WORK_SECONDS":  s.todayWorkSeconds = Int(val) ?? 0
        case "TODAY_BREAK_SECONDS": s.todayBreakSeconds = Int(val) ?? 0
        case "LAST_UPDATE_DATE":  s.lastUpdateDate = val
        default: break
        }
    }
    return s
}

func loadConfig() -> AppConfig {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".config/break-reminder/config.yaml")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppConfig() }

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
        default: break
        }
    }
    return c
}

func launchdStatus() -> String {
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

func getIdleSeconds() -> Int {
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
        // Find HIDIdleTime in nanoseconds
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

// MARK: - Circular Progress View

class CircularProgressView: NSView {
    var progress: CGFloat = 0.0 {
        didSet { needsDisplay = true }
    }
    var trackColor: NSColor = NSColor(white: 0.2, alpha: 1.0)
    var fillColor: NSColor = NSColor(red: 0.3, green: 0.8, blue: 0.5, alpha: 1.0)
    var lineWidth: CGFloat = 10.0

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)

        let center = NSPoint(x: bounds.midX, y: bounds.midY)
        let radius = min(bounds.width, bounds.height) / 2 - lineWidth / 2
        let startAngle: CGFloat = 90
        let endAngle: CGFloat = startAngle - 360

        // Track
        let trackPath = NSBezierPath()
        trackPath.appendArc(withCenter: center, radius: radius, startAngle: startAngle, endAngle: endAngle, clockwise: true)
        trackColor.setStroke()
        trackPath.lineWidth = lineWidth
        trackPath.lineCapStyle = .round
        trackPath.stroke()

        // Fill
        if progress > 0 {
            let fillEnd = startAngle - (360 * progress)
            let fillPath = NSBezierPath()
            fillPath.appendArc(withCenter: center, radius: radius, startAngle: startAngle, endAngle: fillEnd, clockwise: true)
            fillColor.setStroke()
            fillPath.lineWidth = lineWidth
            fillPath.lineCapStyle = .round
            fillPath.stroke()
        }
    }
}

// MARK: - Stat Bar View

class StatBarView: NSView {
    var value: CGFloat = 0.0 {
        didSet { needsDisplay = true }
    }
    var barColor: NSColor = .systemBlue

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)

        let bg = NSBezierPath(roundedRect: bounds, xRadius: 3, yRadius: 3)
        NSColor(white: 0.2, alpha: 1.0).setFill()
        bg.fill()

        if value > 0 {
            let fillWidth = bounds.width * min(value, 1.0)
            let fillRect = NSRect(x: 0, y: 0, width: fillWidth, height: bounds.height)
            let fill = NSBezierPath(roundedRect: fillRect, xRadius: 3, yRadius: 3)
            barColor.setFill()
            fill.fill()
        }
    }
}

// MARK: - Dashboard Window

class DashboardApp: NSObject, NSApplicationDelegate {
    var window: NSWindow!
    var timer: Timer?

    // UI elements
    var statusDot: NSView!
    var statusLabel: NSTextField!
    var modeLabel: NSTextField!
    var circularProgress: CircularProgressView!
    var timeLabel: NSTextField!
    var sessionInfoLabel: NSTextField!

    var dailyWorkLabel: NSTextField!
    var dailyBreakLabel: NSTextField!
    var dailyRatioBar: StatBarView!
    var dailyRatioLabel: NSTextField!

    var systemLabel: NSTextField!
    var idleLabel: NSTextField!

    var resetButton: NSButton!
    var breakButton: NSButton!

    // Colors
    let workColor = NSColor(red: 0.3, green: 0.8, blue: 0.5, alpha: 1.0)
    let breakColor = NSColor(red: 0.4, green: 0.7, blue: 1.0, alpha: 1.0)
    let bgColor = NSColor(red: 0.1, green: 0.1, blue: 0.12, alpha: 1.0)
    let textColor = NSColor(white: 0.9, alpha: 1.0)
    let dimColor = NSColor(white: 0.5, alpha: 1.0)

    func applicationDidFinishLaunching(_ notification: Notification) {
        setupWindow()
        setupUI()
        refresh()

        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.refresh()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }

    func setupWindow() {
        let width: CGFloat = 360
        let height: CGFloat = 520
        let screenFrame = NSScreen.main?.visibleFrame ?? NSRect(x: 0, y: 0, width: 800, height: 600)
        let x = screenFrame.maxX - width - 20
        let y = screenFrame.maxY - height - 20

        window = NSWindow(
            contentRect: NSRect(x: x, y: y, width: width, height: height),
            styleMask: [.titled, .closable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Break Reminder"
        window.backgroundColor = bgColor
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .hidden
        window.isMovableByWindowBackground = true
        window.level = .floating
        window.makeKeyAndOrderFront(nil)

        NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            switch event.charactersIgnoringModifiers {
            case "q": NSApp.terminate(nil); return nil
            case "r": self?.resetTimer(); return nil
            case "b": self?.forceBreak(); return nil
            default: return event
            }
        }
    }

    func setupUI() {
        let content = window.contentView!
        var y: CGFloat = 470

        // ── Header ──
        let title = makeLabel("Break Reminder", size: 20, weight: .bold, color: textColor)
        title.frame = NSRect(x: 20, y: y, width: 320, height: 28)
        content.addSubview(title)
        y -= 8

        let hint = makeLabel("q: quit  r: reset  b: break", size: 11, weight: .regular, color: dimColor)
        hint.frame = NSRect(x: 20, y: y, width: 320, height: 16)
        content.addSubview(hint)
        y -= 30

        // ── Status Row ──
        statusDot = NSView(frame: NSRect(x: 20, y: y + 4, width: 12, height: 12))
        statusDot.wantsLayer = true
        statusDot.layer?.cornerRadius = 6
        statusDot.layer?.backgroundColor = workColor.cgColor
        content.addSubview(statusDot)

        statusLabel = makeLabel("WORKING", size: 16, weight: .semibold, color: workColor)
        statusLabel.frame = NSRect(x: 40, y: y, width: 120, height: 22)
        content.addSubview(statusLabel)

        modeLabel = makeLabel("", size: 12, weight: .regular, color: dimColor)
        modeLabel.alignment = .right
        modeLabel.frame = NSRect(x: 160, y: y + 2, width: 180, height: 18)
        content.addSubview(modeLabel)
        y -= 20

        // ── Divider ──
        let div1 = makeDivider(y: y)
        content.addSubview(div1)
        y -= 16

        // ── Circular Progress ──
        let ringSize: CGFloat = 160
        let ringX = (360 - ringSize) / 2
        circularProgress = CircularProgressView(frame: NSRect(x: ringX, y: y - ringSize, width: ringSize, height: ringSize))
        circularProgress.fillColor = workColor
        content.addSubview(circularProgress)

        timeLabel = makeLabel("00:00", size: 36, weight: .ultraLight, color: textColor)
        timeLabel.alignment = .center
        timeLabel.font = NSFont.monospacedDigitSystemFont(ofSize: 36, weight: .ultraLight)
        timeLabel.frame = NSRect(x: ringX, y: y - ringSize / 2 - 12, width: ringSize, height: 44)
        content.addSubview(timeLabel)

        sessionInfoLabel = makeLabel("", size: 12, weight: .regular, color: dimColor)
        sessionInfoLabel.alignment = .center
        sessionInfoLabel.frame = NSRect(x: ringX, y: y - ringSize / 2 - 32, width: ringSize, height: 16)
        content.addSubview(sessionInfoLabel)

        y -= ringSize + 20

        // ── Divider ──
        let div2 = makeDivider(y: y)
        content.addSubview(div2)
        y -= 20

        // ── Daily Stats ──
        let statsTitle = makeLabel("Daily Statistics", size: 14, weight: .semibold, color: textColor)
        statsTitle.frame = NSRect(x: 20, y: y, width: 320, height: 20)
        content.addSubview(statsTitle)
        y -= 24

        dailyWorkLabel = makeLabel("Work: 0 min", size: 13, weight: .regular, color: textColor)
        dailyWorkLabel.frame = NSRect(x: 20, y: y, width: 150, height: 18)
        content.addSubview(dailyWorkLabel)

        dailyBreakLabel = makeLabel("Break: 0 min", size: 13, weight: .regular, color: breakColor)
        dailyBreakLabel.alignment = .right
        dailyBreakLabel.frame = NSRect(x: 190, y: y, width: 150, height: 18)
        content.addSubview(dailyBreakLabel)
        y -= 22

        dailyRatioBar = StatBarView(frame: NSRect(x: 20, y: y, width: 260, height: 8))
        dailyRatioBar.barColor = workColor
        content.addSubview(dailyRatioBar)

        dailyRatioLabel = makeLabel("", size: 11, weight: .regular, color: dimColor)
        dailyRatioLabel.frame = NSRect(x: 288, y: y - 3, width: 52, height: 14)
        content.addSubview(dailyRatioLabel)
        y -= 24

        // ── Divider ──
        let div3 = makeDivider(y: y)
        content.addSubview(div3)
        y -= 20

        // ── System Info ──
        systemLabel = makeLabel("System: ...", size: 12, weight: .regular, color: dimColor)
        systemLabel.frame = NSRect(x: 20, y: y, width: 320, height: 16)
        content.addSubview(systemLabel)
        y -= 18

        idleLabel = makeLabel("Idle: 0s", size: 12, weight: .regular, color: dimColor)
        idleLabel.frame = NSRect(x: 20, y: y, width: 320, height: 16)
        content.addSubview(idleLabel)
        y -= 28

        // ── Buttons ──
        resetButton = makeStyledButton(title: "Reset", action: #selector(resetTimer))
        resetButton.frame = NSRect(x: 40, y: y, width: 120, height: 32)
        content.addSubview(resetButton)

        breakButton = makeStyledButton(title: "Force Break", action: #selector(forceBreak))
        breakButton.frame = NSRect(x: 200, y: y, width: 120, height: 32)
        content.addSubview(breakButton)
    }

    func refresh() {
        let state = loadState()
        let config = loadConfig()
        let now = Int64(Date().timeIntervalSince1970)
        let idleSec = getIdleSeconds()

        let isWork = state.mode == "work"
        let activeColor = isWork ? workColor : breakColor

        // Status
        statusDot.layer?.backgroundColor = activeColor.cgColor
        statusLabel.stringValue = isWork ? "WORKING" : "ON BREAK"
        statusLabel.textColor = activeColor

        // Progress — interpolate between check intervals for smooth updates
        var progress: CGFloat = 0
        var remaining: Int = 0
        var totalSec: Int = 0
        var elapsedSec: Int = 0

        if isWork {
            totalSec = config.workDurationMin * 60
            // workSeconds from file + time since last check = real-time estimate
            let sinceLastCheck = max(Int(now - state.lastCheck), 0)
            elapsedSec = state.workSeconds + sinceLastCheck
            if totalSec > 0 {
                progress = CGFloat(elapsedSec) / CGFloat(totalSec)
            }
            remaining = max(totalSec - elapsedSec, 0)
            modeLabel.stringValue = "\(elapsedSec / 60) / \(config.workDurationMin) min"
            sessionInfoLabel.stringValue = "until break"
        } else {
            totalSec = config.breakDurationMin * 60
            elapsedSec = Int(now - state.breakStart)
            if totalSec > 0 {
                progress = CGFloat(elapsedSec) / CGFloat(totalSec)
            }
            remaining = max(totalSec - elapsedSec, 0)
            modeLabel.stringValue = "\(elapsedSec / 60) / \(config.breakDurationMin) min"
            sessionInfoLabel.stringValue = "until work"
        }

        circularProgress.progress = min(progress, 1.0)
        circularProgress.fillColor = activeColor
        circularProgress.needsDisplay = true

        let min = remaining / 60
        let sec = remaining % 60
        timeLabel.stringValue = String(format: "%02d:%02d", min, sec)

        // Daily stats — also interpolate work seconds
        let dailyWorkSec = isWork
            ? state.todayWorkSeconds + max(Int(now - state.lastCheck), 0)
            : state.todayWorkSeconds
        let workMin = dailyWorkSec / 60
        let breakMin = state.todayBreakSeconds / 60
        dailyWorkLabel.stringValue = "Work: \(workMin) min"
        dailyBreakLabel.stringValue = "Break: \(breakMin) min"

        let total = workMin + breakMin
        if total > 0 {
            let ratio = CGFloat(workMin) / CGFloat(total)
            dailyRatioBar.value = ratio
            dailyRatioBar.needsDisplay = true
            dailyRatioLabel.stringValue = "\(Int(ratio * 100))%"
        } else {
            dailyRatioBar.value = 0
            dailyRatioBar.needsDisplay = true
            dailyRatioLabel.stringValue = "-"
        }

        // System
        systemLabel.stringValue = "System: \(launchdStatus())"
        idleLabel.stringValue = "Idle: \(idleSec)s / Threshold: \(config.idleThresholdSec)s"
    }

    @objc func resetTimer() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let path = home.appendingPathComponent(".break-reminder-state")
        let now = Int64(Date().timeIntervalSince1970)
        let df = DateFormatter()
        df.dateFormat = "yyyy-MM-dd"
        let today = df.string(from: Date())
        let lines = [
            "WORK_SECONDS=0",
            "MODE=work",
            "LAST_CHECK=\(now)",
            "BREAK_START=0",
            "TODAY_WORK_SECONDS=0",
            "TODAY_BREAK_SECONDS=0",
            "LAST_UPDATE_DATE=\(today)",
        ]
        try? lines.joined(separator: "\n").data(using: .utf8)?.write(to: path)
        refresh()
    }

    @objc func forceBreak() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let path = home.appendingPathComponent(".break-reminder-state")
        let state = loadState()
        let now = Int64(Date().timeIntervalSince1970)
        let lines = [
            "WORK_SECONDS=0",
            "MODE=break",
            "LAST_CHECK=\(now)",
            "BREAK_START=\(now)",
            "TODAY_WORK_SECONDS=\(state.todayWorkSeconds)",
            "TODAY_BREAK_SECONDS=\(state.todayBreakSeconds)",
            "LAST_UPDATE_DATE=\(state.lastUpdateDate)",
        ]
        try? lines.joined(separator: "\n").data(using: .utf8)?.write(to: path)
        refresh()
    }

    // MARK: - Helpers

    func makeLabel(_ text: String, size: CGFloat, weight: NSFont.Weight, color: NSColor) -> NSTextField {
        let label = NSTextField(labelWithString: text)
        label.font = NSFont.systemFont(ofSize: size, weight: weight)
        label.textColor = color
        label.drawsBackground = false
        label.isBezeled = false
        label.isEditable = false
        return label
    }

    func makeStyledButton(title: String, action: Selector) -> NSButton {
        let button = NSButton(title: title, target: self, action: action)
        button.bezelStyle = .rounded
        button.isBordered = false
        button.wantsLayer = true
        button.layer?.backgroundColor = NSColor(white: 0.22, alpha: 1.0).cgColor
        button.layer?.cornerRadius = 8
        button.contentTintColor = NSColor(white: 0.9, alpha: 1.0)
        button.font = NSFont.systemFont(ofSize: 14, weight: .medium)
        return button
    }

    func makeDivider(y: CGFloat) -> NSView {
        let div = NSView(frame: NSRect(x: 20, y: y, width: 320, height: 1))
        div.wantsLayer = true
        div.layer?.backgroundColor = NSColor(white: 0.2, alpha: 1.0).cgColor
        return div
    }
}

// MARK: - Main

let app = NSApplication.shared
let delegate = DashboardApp()
app.delegate = delegate
app.setActivationPolicy(.regular)
app.activate(ignoringOtherApps: true)
app.run()
