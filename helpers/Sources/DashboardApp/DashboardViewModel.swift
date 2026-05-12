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
