import Foundation
import SwiftUI
import HelperCore

enum DashboardTab: String, CaseIterable, Identifiable {
    case timer = "타이머"
    case stats = "통계"
    case insights = "인사이트"
    case settings = "설정"

    var id: String { rawValue }
}

@MainActor
final class DashboardViewModel: ObservableObject {
    @Published var state: AppState = AppState()
    @Published var config: AppConfig = AppConfig()
    @Published var idleSeconds: Int = 0
    @Published var launchdStatusText: String = "Unknown"
    @Published var selectedTab: DashboardTab = .timer
    @Published var history: [HistoryEntry] = []
    @Published var insights: InsightsReport?
    @Published var isRefreshingInsights = false
    @Published var showConfetti = false
    private var lastGoalCheckMinute = 0
    private let dailyGoalMinutes = 240 // 4 hours

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

    var currentMascot: Mascot {
        mascotFor(state: state, config: config, now: now)
    }

    var statusText: String {
        if isPaused {
            let label = pauseModeLabel
            return label.isEmpty ? "PAUSED" : "PAUSED · \(label)"
        }
        return isWork ? "WORKING" : "ON BREAK"
    }

    var pauseModeLabel: String {
        switch state.pauseReason {
        case "meeting": return "회의"
        case "focus":   return "집중"
        case "afk":     return "외출"
        default:        return ""
        }
    }

    var pauseRemainingText: String? {
        guard isPaused, state.pauseUntil > 0 else { return nil }
        let remaining = max(0, state.pauseUntil - now)
        let m = remaining / 60
        let s = remaining % 60
        return String(format: "%d:%02d 남음", m, s)
    }

    var pauseModeAccent: Color {
        switch state.pauseReason {
        case "meeting": return .blue
        case "focus":   return .orange
        case "afk":     return .gray
        default:        return .secondary
        }
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
        loadHistory()
        loadInsights()
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
        loadHistory()
        loadInsights()
        checkGoalAchievement()
    }

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

    func loadHistory() {
        history = loadHistoryFromDisk()
    }

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

        dashLog("insights refresh: requested")

        var checked: [String] = []
        guard let cli = findHelper("break-reminder", checked: &checked) else {
            dashLog("insights refresh: break-reminder helper not found. checked=\(checked)")
            return
        }
        dashLog("insights refresh: helper=\(cli) PATH=\(ProcessInfo.processInfo.environment["PATH"] ?? "<unset>")")

        let process = Process()
        process.launchPath = cli
        process.arguments = ["insights", "--refresh"]
        let outPipe = Pipe()
        let errPipe = Pipe()
        process.standardOutput = outPipe
        process.standardError = errPipe

        do {
            try process.run()
        } catch {
            dashLog("insights refresh: spawn failed: \(error.localizedDescription)")
            return
        }
        process.waitUntilExit()

        let outData = outPipe.fileHandleForReading.readDataToEndOfFile()
        let errData = errPipe.fileHandleForReading.readDataToEndOfFile()
        let outStr = String(data: outData, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let errStr = String(data: errData, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

        dashLog("insights refresh: exit=\(process.terminationStatus) reason=\(process.terminationReason.rawValue) stdoutBytes=\(outData.count) stderrBytes=\(errData.count)")
        if !outStr.isEmpty {
            dashLog("insights refresh: stdout: \(truncated(outStr, max: 800))")
        }
        if !errStr.isEmpty {
            dashLog("insights refresh: stderr: \(truncated(errStr, max: 800))")
        }

        loadInsights()
        if insights == nil {
            dashLog("insights refresh: insights file still empty after run")
        } else {
            dashLog("insights refresh: insights loaded ok")
        }
    }

    private func truncated(_ s: String, max: Int) -> String {
        if s.count <= max { return s }
        return String(s.prefix(max)) + "…(+\(s.count - max) chars)"
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

    func pause(mode: String, durationMinutes: Int? = nil) {
        var args = ["pause", "--mode=\(mode)"]
        if let m = durationMinutes, m > 0 {
            args.append("--duration=\(m)m")
        }
        runCLI(args: args)
        refresh()
    }

    func resume() {
        runCLI(args: ["resume"])
        refresh()
    }

    private func runCLI(args: [String]) {
        guard let cli = findHelper("break-reminder") else { return }
        let process = Process()
        process.launchPath = cli
        process.arguments = args
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice
        do {
            try process.run()
            process.waitUntilExit()
        } catch {
            return
        }
    }

    func saveSettings(_ changes: [(String, String)]) -> Result<Void, Error> {
        guard let cli = findHelper("break-reminder") else {
            return .failure(NSError(domain: "ConfigSave", code: 1,
                userInfo: [NSLocalizedDescriptionKey: "break-reminder CLI not found"]))
        }
        let args = ["config", "set"] + changes.map { "\($0.0)=\($0.1)" }
        let process = Process()
        process.launchPath = cli
        process.arguments = args
        let errPipe = Pipe()
        process.standardOutput = FileHandle.nullDevice
        process.standardError = errPipe

        do {
            try process.run()
            process.waitUntilExit()
            if process.terminationStatus != 0 {
                let msg = String(data: errPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8)
                    ?? "Unknown error"
                return .failure(NSError(domain: "ConfigSave", code: Int(process.terminationStatus),
                    userInfo: [NSLocalizedDescriptionKey: msg.trimmingCharacters(in: .whitespacesAndNewlines)]))
            }
            refresh()
            return .success(())
        } catch {
            return .failure(error)
        }
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
