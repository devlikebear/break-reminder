import Foundation
import Combine

public func pomodoroStartCommands(state: AppState, config: AppConfig) -> [[String]] {
    guard !state.paused, state.mode == "work" else { return [] }
    if config.pomodoroEnabled {
        return state.isSessionActive ? [] : [["start"]]
    }
    return state.isSessionActive ? [["pomodoro", "on"]] : [["pomodoro", "on"], ["start"]]
}

public func pomodoroStatus(state: AppState, config: AppConfig) -> String {
    guard config.pomodoroEnabled else { return "기본 주기 \(config.workDurationMin)/\(config.breakDurationMin)분" }
    let every = max(1, config.pomodoroLongBreakEvery)
    let isLongBreak = state.mode == "break" && state.pomodoroCount > 0 && state.pomodoroCount % every == 0
    let phase = state.mode == "break" ? (isLongBreak ? "긴 휴식" : "짧은 휴식") : "집중 \(state.pomodoroCount % every + 1)/\(every)"
    return "\(phase) · 오늘 완료 \(state.todayPomodoros)회"
}

@MainActor
public final class PomodoroActions: ObservableObject {
    @Published public private(set) var isBusy = false
    @Published public private(set) var message: String?
    public init() {}

    public func perform(commands: [[String]], client: CLICommandRunning) async {
        guard !isBusy else { return }
        isBusy = true; message = nil
        defer { isBusy = false }
        var modeSaved = false
        for args in commands {
            let result = await client.run(arguments: args)
            guard result.succeeded else {
                message = (modeSaved ? "포모도로 설정은 저장됐지만 작업 시작에 실패했습니다. " : "") + result.errorMessage
                return
            }
            if args == ["pomodoro", "on"] { modeSaved = true }
        }
    }
}
