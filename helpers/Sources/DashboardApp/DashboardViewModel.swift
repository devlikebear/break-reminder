import Foundation
import SwiftUI
import HelperCore

enum DashboardTab: String, CaseIterable, Identifiable {
    case timer = "Timer"
    case stats = "Statistics"
    case insights = "Insights"
    case settings = "Settings"

    var id: String { rawValue }
}

enum InsightsRefreshStatus: Equatable {
    case idle
    case running
    case succeeded
    case failed(String)
}

@MainActor
final class DashboardViewModel: ObservableObject {
    @Published var state: AppState = AppState()
    @Published var config: AppConfig = AppConfig()
    @Published var idleSeconds: Int = 0
    @Published var launchdStatusText: String = L10n.text("Unknown")
    @Published var selectedTimeTool = ProcessInfo.processInfo.arguments.contains("--time-tools") ? "countdown" : "focus"
    @Published var selectedTab: DashboardTab = .timer
    @Published var history: [HistoryEntry] = []
    @Published var insights: InsightsReport?
    @Published private(set) var insightsRefreshStatus: InsightsRefreshStatus = .idle
    @Published var showConfetti = false
    /// Version of the installed CLI, resolved once in `start()`.
    @Published private(set) var appVersion: String = AboutInfo.unknownVersion
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

    var isSessionRunning: Bool { timerIsRunning(state: state, config: config) }

    var isRefreshingInsights: Bool {
        if case .running = insightsRefreshStatus {
            return true
        }
        return false
    }

    var statusText: String {
        if isPaused {
            let label = pauseModeLabel
            return label.isEmpty ? L10n.text("PAUSED") : L10n.text("PAUSED · {0}", label)
        }
        if !isSessionRunning {
            return state.sessionManual ? L10n.text("STOPPED") : L10n.text("IDLE")
        }
        return isWork ? L10n.text("WORKING") : L10n.text("ON BREAK")
    }

    var pauseModeLabel: String {
        switch state.pauseReason {
        case "meeting": return L10n.text("Meeting")
        case "focus":   return L10n.text("Focus")
        case "afk":     return L10n.text("Away")
        default:        return ""
        }
    }

    var pauseRemainingText: String? {
        guard isPaused, state.pauseUntil > 0 else { return nil }
        let remaining = max(0, state.pauseUntil - now)
        let m = remaining / 60
        let s = remaining % 60
        return String(format: L10n.text("%d:%02d remaining"), m, s)
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
            return L10n.text("{0} / {1} min", sp.elapsedSec / 60, config.effectiveWorkMin)
        } else {
            return L10n.text("{0} / {1} min", sp.elapsedSec / 60, config.effectiveBreakMin(completedPomodoros: state.pomodoroCount))
        }
    }

    var sessionSubtitle: String {
        if isPaused { return L10n.text("paused") }
        return isWork ? L10n.text("until break") : L10n.text("until work")
    }

    func start() {
        appVersion = queryInstalledVersion()
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
        insightsRefreshStatus = .running

        dashLog("insights refresh: requested")

        var checked: [String] = []
        guard let cli = findHelper("break-reminder", checked: &checked) else {
            let message = L10n.text("Could not find break-reminder CLI.\nSearched paths:\n{0}\nPlace the dashboard and CLI in the same installation directory or run `make install`.", checked.map { "• \($0)" }.joined(separator: "\n"))
            insightsRefreshStatus = .failed(message)
            dashLog("insights refresh: \(message)")
            return
        }

        let environment = helperProcessEnvironment()
        dashLog("insights refresh: helper=\(cli) PATH=\(environment["PATH"] ?? "<unset>")")

        // Process.waitUntilExit() must stay off the main actor. Otherwise the
        // spinner cannot render and the dashboard appears to ignore the click.
        Task { [weak self] in
            let result = await Task.detached(priority: .userInitiated) {
                executeInsightsRefresh(cliPath: cli, environment: environment)
            }.value
            self?.completeInsightsRefresh(result)
        }
    }

    private func completeInsightsRefresh(_ result: InsightsProcessResult) {
        dashLog("insights refresh: exit=\(result.terminationStatus) reason=\(result.terminationReason) outputBytes=\(result.output.utf8.count)")
        if !result.output.isEmpty {
            dashLog("insights refresh: output: \(truncated(result.output, max: 800))")
        }

        if result.launchError != nil || result.terminationStatus != 0 {
            let message = insightsRefreshFailureMessage(result)
            insightsRefreshStatus = .failed(message)
            dashLog("insights refresh: failed: \(message)")
            return
        }

        loadInsights()
        if insights == nil {
            let message = L10n.text("The CLI exited successfully, but the insights report could not be read.\nFile: ~/.break-reminder-insights.json\nCLI output:\n{0}\nCheck that the file exists and its JSON format and permissions are correct.", result.output.isEmpty ? L10n.text("None") : truncated(result.output, max: 4000))
            insightsRefreshStatus = .failed(message)
            dashLog("insights refresh: \(message)")
            return
        }

        insightsRefreshStatus = .succeeded
        dashLog("insights refresh: insights loaded ok")
    }

    private func insightsRefreshFailureMessage(_ result: InsightsProcessResult) -> String {
        var lines = [
            L10n.text("Could not run AI analysis."),
            L10n.text("Executable: {0}", result.cliPath),
            L10n.text("Command: insights --refresh"),
        ]

        if let launchError = result.launchError {
            lines.append(L10n.text("Process launch failed: {0}", launchError))
        } else {
            lines.append(L10n.text("Exit code: {0}", result.terminationStatus))
            lines.append(L10n.text("Exit reason: {0}", result.terminationReason))
            if result.output.isEmpty {
                lines.append(L10n.text("The CLI did not return a detailed error message."))
            } else {
                lines.append(L10n.text("CLI output:\n{0}", truncated(result.output, max: 4000)))
            }
        }
        lines.append(L10n.text("Execution PATH: {0}", result.environmentPath))
        lines.append(L10n.text("Detailed log: ~/.break-reminder.log"))
        return lines.joined(separator: "\n")
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
        s.todayPomodoros = state.todayPomodoros
        carryOverSession(into: &s)
        writeStateToDisk(s)
        refresh()
    }

    /// Dashboard writes rebuild the state file from scratch, so the work-session
    /// fields owned by the Go timer have to be carried over explicitly.
    private func carryOverSession(into next: inout AppState) {
        next.sessionState = state.sessionState
        next.sessionStart = state.sessionStart
        next.sessionEnd = state.sessionEnd
        next.sessionManual = state.sessionManual
        next.lastActivity = state.lastActivity
    }

    func toggleSession() {
        runCLI(args: [isSessionRunning ? "stop" : "start"])
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
                userInfo: [NSLocalizedDescriptionKey: L10n.text("break-reminder CLI not found")]))
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
                    ?? L10n.text("Unknown error")
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
        s.pomodoroCount = state.pomodoroCount
        s.todayPomodoros = state.todayPomodoros
        carryOverSession(into: &s)
        writeStateToDisk(s)
        refresh()
    }
}

private struct InsightsProcessResult: Sendable {
    let cliPath: String
    let environmentPath: String
    let terminationStatus: Int32
    let terminationReason: String
    let output: String
    let launchError: String?
}

private func executeInsightsRefresh(cliPath: String, environment: [String: String]) -> InsightsProcessResult {
    let process = Process()
    process.executableURL = URL(fileURLWithPath: cliPath)
    process.arguments = ["insights", "--refresh"]
    process.environment = environment

    // Merge stdout and stderr into one draining pipe. Reading before waiting
    // prevents a verbose AI CLI from filling the pipe and hanging the refresh.
    let outputPipe = Pipe()
    process.standardOutput = outputPipe
    process.standardError = outputPipe

    do {
        try process.run()
    } catch {
        return InsightsProcessResult(
            cliPath: cliPath,
            environmentPath: environment["PATH"] ?? "<unset>",
            terminationStatus: -1,
            terminationReason: L10n.text("not started"),
            output: "",
            launchError: error.localizedDescription
        )
    }

    let outputData = outputPipe.fileHandleForReading.readDataToEndOfFile()
    process.waitUntilExit()
    let output = String(data: outputData, encoding: .utf8)?
        .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

    return InsightsProcessResult(
        cliPath: cliPath,
        environmentPath: environment["PATH"] ?? "<unset>",
        terminationStatus: process.terminationStatus,
        terminationReason: process.terminationReason == .exit ? L10n.text("normal exit") : L10n.text("uncaught signal"),
        output: output,
        launchError: nil
    )
}
