import Foundation

public struct TodayTotals: Equatable {
    public let workSeconds: Int
    public let breakSeconds: Int
    public let date: String

    public var workMinutes: Int { workSeconds / 60 }
    public var breakMinutes: Int { breakSeconds / 60 }
}

public struct MenuBarPresentation: Equatable {
    public let title: String
    public let statusLine: String
    public let statsLine: String
}

public func todayTotals(state: AppState, config: AppConfig, now: Int64) -> TodayTotals {
    let live = liveDailyTotals(state: state, config: config, now: now)
    return TodayTotals(
        workSeconds: live.workSeconds,
        breakSeconds: live.breakSeconds,
        date: live.date
    )
}

public func menuBarPresentation(state: AppState, config: AppConfig, now: Int64) -> MenuBarPresentation {
    let totals = todayTotals(state: state, config: config, now: now)

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
