import Foundation

public struct TodayTotals: Equatable {
    public let workSeconds: Int
    public let breakSeconds: Int

    public var workMinutes: Int { workSeconds / 60 }
    public var breakMinutes: Int { breakSeconds / 60 }
}

public struct MenuBarPresentation: Equatable {
    public let title: String
    public let statusLine: String
    public let statsLine: String
}

public func todayTotals(state: AppState, now: Int64) -> TodayTotals {
    let sinceLastCheck = max(Int(now - state.lastCheck), 0)
    let interpolatedWorkSeconds = state.mode == "work"
        ? state.todayWorkSeconds + sinceLastCheck
        : state.todayWorkSeconds
    let interpolatedBreakSeconds = state.mode == "break"
        ? state.todayBreakSeconds + sinceLastCheck
        : state.todayBreakSeconds

    return TodayTotals(
        workSeconds: max(interpolatedWorkSeconds, 0),
        breakSeconds: max(interpolatedBreakSeconds, 0)
    )
}

public func menuBarPresentation(state: AppState, config: AppConfig, now: Int64) -> MenuBarPresentation {
    let totals = todayTotals(state: state, now: now)

    if state.mode == "break" {
        let progress = breakProgress(state: state, config: config, now: now)
        let percent = Int(progress.progress * 100)
        let elapsedMinutes = progress.elapsedSec / 60
        let remainingMinutes = progress.remainingSec / 60

        return MenuBarPresentation(
            title: "☕ \(percent)% · \(remainingMinutes)m left",
            statusLine: "On break · \(elapsedMinutes)m elapsed · \(remainingMinutes)m until work",
            statsLine: "Today · Work \(formatMinutes(totals.workMinutes)) · Break \(formatMinutes(totals.breakMinutes))"
        )
    }

    let progress = workProgress(state: state, config: config, now: now)
    let percent = Int(progress.progress * 100)
    let elapsedMinutes = progress.elapsedSec / 60
    let remainingMinutes = progress.remainingSec / 60

    return MenuBarPresentation(
        title: "🟢 \(percent)% · \(remainingMinutes)m left",
        statusLine: "Working · \(elapsedMinutes)m elapsed · \(remainingMinutes)m until break",
        statsLine: "Today · Work \(formatMinutes(totals.workMinutes)) · Break \(formatMinutes(totals.breakMinutes))"
    )
}
