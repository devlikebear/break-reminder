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
    guard config.pomodoroEnabled else { return L10n.text("Classic cycle {0}/{1} min", config.workDurationMin, config.breakDurationMin) }
    let every = max(1, config.pomodoroLongBreakEvery)
    let isLongBreak = state.mode == "break" && state.pomodoroCount > 0 && state.pomodoroCount % every == 0
    let phase = state.mode == "break" ? (isLongBreak ? L10n.text("Long break") : L10n.text("Short break")) : L10n.text("Focus {0}/{1}", state.pomodoroCount % every + 1, every)
    return L10n.text("{0} · completed today: {1}", phase, state.todayPomodoros)
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
                message = (modeSaved ? L10n.text("Pomodoro settings were saved, but starting work failed. ") : "") + result.errorMessage
                return
            }
            if args == ["pomodoro", "on"] { modeSaved = true }
        }
    }
}
