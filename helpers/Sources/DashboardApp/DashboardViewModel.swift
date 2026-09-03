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
    @Published var launchdStatusText: String = "Unknown"
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
            return label.isEmpty ? "PAUSED" : "PAUSED · \(label)"
        }
        if !isSessionRunning {
            return state.sessionManual ? "STOPPED" : "IDLE"
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
            let message = """
            break-reminder CLI를 찾지 못했습니다.
            확인한 경로:
            \(checked.map { "• \($0)" }.joined(separator: "\n"))
            대시보드와 CLI를 같은 설치 위치에 두거나 `make install` 후 다시 시도하세요.
            """
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
            let message = """
            CLI는 종료 코드 0으로 완료했지만 인사이트 리포트를 읽지 못했습니다.
            파일: ~/.break-reminder-insights.json
            CLI 출력:
            \(result.output.isEmpty ? "(없음)" : truncated(result.output, max: 4000))
            파일이 생성되었는지와 JSON 형식/권한을 확인한 뒤 다시 시도하세요.
            """
            insightsRefreshStatus = .failed(message)
            dashLog("insights refresh: \(message)")
            return
        }

        insightsRefreshStatus = .succeeded
        dashLog("insights refresh: insights loaded ok")
    }

    private func insightsRefreshFailureMessage(_ result: InsightsProcessResult) -> String {
        var lines = [
            "AI 분석 명령을 실행하지 못했습니다.",
            "실행 파일: \(result.cliPath)",
            "명령: insights --refresh",
        ]

        if let launchError = result.launchError {
            lines.append("프로세스 시작 실패: \(launchError)")
        } else {
            lines.append("종료 코드: \(result.terminationStatus)")
            lines.append("종료 사유: \(result.terminationReason)")
            if result.output.isEmpty {
                lines.append("CLI가 상세 오류 메시지를 반환하지 않았습니다.")
            } else {
                lines.append("CLI 출력:\n\(truncated(result.output, max: 4000))")
            }
        }
        lines.append("실행 PATH: \(result.environmentPath)")
        lines.append("상세 로그: ~/.break-reminder.log")
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
            terminationReason: "not started",
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
        terminationReason: process.terminationReason == .exit ? "normal exit" : "uncaught signal",
        output: output,
        launchError: nil
    )
}
