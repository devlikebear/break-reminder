import Foundation
import SwiftUI
import HelperCore

enum DashboardTab: String, CaseIterable, Identifiable {
    case timer = "타이머"
    case stats = "통계"
    case insights = "인사이트"

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
