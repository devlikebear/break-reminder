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
    let statsLine = L10n.text("Today · Work {0} · Break {1}", formatMinutes(totals.workMinutes), formatMinutes(totals.breakMinutes))

    if !state.paused && !timerIsRunning(state: state, config: config) {
        return MenuBarPresentation(
            title: sessionOffTitle(state: state),
            statusLine: sessionOffStatusLine(state: state),
            statsLine: statsLine
        )
    }

    if state.mode == "break" {
        let progress = breakProgress(state: state, config: config, now: now)
        let percent = Int(progress.progress * 100)
        let elapsedMinutes = progress.elapsedSec / 60
        let remainingMinutes = progress.remainingSec / 60

        if state.paused {
            return MenuBarPresentation(
                title: L10n.text("PAUSED (BREAK) · {0}m left", remainingMinutes),
                statusLine: L10n.text("PAUSED (BREAK) · {0}m elapsed · {1}m until work", elapsedMinutes, remainingMinutes),
                statsLine: statsLine
            )
        }

        return MenuBarPresentation(
            title: L10n.text("{0}% · {1}m left", percent, remainingMinutes),
            statusLine: L10n.text("On break · {0}m elapsed · {1}m until work", elapsedMinutes, remainingMinutes),
            statsLine: statsLine
        )
    }

    let progress = workProgress(state: state, config: config, now: now)
    let percent = Int(progress.progress * 100)
    let elapsedMinutes = progress.elapsedSec / 60
    let remainingMinutes = progress.remainingSec / 60

    if state.paused {
        return MenuBarPresentation(
            title: L10n.text("PAUSED (WORK) · {0}m left", remainingMinutes),
            statusLine: L10n.text("PAUSED (WORK) · {0}m elapsed · {1}m until break", elapsedMinutes, remainingMinutes),
            statsLine: statsLine
        )
    }

    return MenuBarPresentation(
        title: L10n.text("{0}% · {1}m left", percent, remainingMinutes),
        statusLine: L10n.text("Working · {0}m elapsed · {1}m until break", elapsedMinutes, remainingMinutes),
        statsLine: statsLine
    )
}
