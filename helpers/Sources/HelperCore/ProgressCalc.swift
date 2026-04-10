import Foundation

/// Calculates session progress and remaining time.
public struct SessionProgress {
    public let progress: Double   // 0.0 ~ 1.0
    public let remainingSec: Int
    public let elapsedSec: Int
    public let totalSec: Int

    public var remainingFormatted: String { formatTime(remainingSec) }
}

public struct LiveDailyTotals: Equatable {
    public let workSeconds: Int
    public let breakSeconds: Int
    public let date: String
}

private func currentDayString(now: Int64) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))
}

private func startOfCurrentDayUnix(now: Int64) -> Int64 {
    let date = Date(timeIntervalSince1970: TimeInterval(now))
    return Int64(Calendar.current.startOfDay(for: date).timeIntervalSince1970)
}

private func maxExpectedElapsed(config: AppConfig) -> Int {
    max(config.checkIntervalSec * 3, 300)
}

private func activeElapsedSinceLastCheck(state: AppState, config: AppConfig, now: Int64) -> Int {
    guard !state.paused else { return 0 }
    guard state.lastCheck > 0 else { return 0 }
    let elapsed = max(Int(now - state.lastCheck), 0)
    guard elapsed <= maxExpectedElapsed(config: config) else { return 0 }
    return elapsed
}

private func currentDayActiveElapsed(state: AppState, config: AppConfig, now: Int64) -> Int {
    guard !state.paused else { return 0 }
    guard state.lastCheck > 0 else { return 0 }
    let elapsed = max(Int(now - state.lastCheck), 0)
    guard elapsed <= maxExpectedElapsed(config: config) else { return 0 }
    let effectiveStart = max(state.lastCheck, startOfCurrentDayUnix(now: now))
    return max(Int(now - effectiveStart), 0)
}

/// Returns the currently elapsed work seconds, only interpolating when the
/// state has a valid last-check timestamp and recent tick window.
public func liveWorkSeconds(state: AppState, config: AppConfig, now: Int64) -> Int {
    return max(state.workSeconds, 0) + activeElapsedSinceLastCheck(state: state, config: config, now: now)
}

/// Returns today's work total with day-rollover-aware interpolation.
public func liveTodayWorkSeconds(state: AppState, config: AppConfig, now: Int64) -> Int {
    let today = currentDayString(now: now)
    let base = state.lastUpdateDate == today ? max(state.todayWorkSeconds, 0) : 0
    guard state.mode == "work" else { return base }

    let liveDelta = state.lastUpdateDate == today
        ? activeElapsedSinceLastCheck(state: state, config: config, now: now)
        : currentDayActiveElapsed(state: state, config: config, now: now)
    return base + liveDelta
}

/// Returns today's break total with day-rollover-aware interpolation.
public func liveTodayBreakSeconds(state: AppState, config: AppConfig, now: Int64) -> Int {
    let today = currentDayString(now: now)
    let base = state.lastUpdateDate == today ? max(state.todayBreakSeconds, 0) : 0
    guard state.mode == "break" else { return base }

    let liveDelta = state.lastUpdateDate == today
        ? activeElapsedSinceLastCheck(state: state, config: config, now: now)
        : currentDayActiveElapsed(state: state, config: config, now: now)
    return base + liveDelta
}

/// Normalizes today's totals for UI-triggered state writes so they stay on the
/// correct calendar day while preserving the latest visible totals.
public func liveDailyTotals(state: AppState, config: AppConfig, now: Int64) -> LiveDailyTotals {
    LiveDailyTotals(
        workSeconds: liveTodayWorkSeconds(state: state, config: config, now: now),
        breakSeconds: liveTodayBreakSeconds(state: state, config: config, now: now),
        date: currentDayString(now: now)
    )
}

/// Calculates work session progress, interpolating between check intervals.
public func workProgress(state: AppState, config: AppConfig, now: Int64) -> SessionProgress {
    let totalSec = config.workDurationMin * 60
    let elapsed = liveWorkSeconds(state: state, config: config, now: now)
    let remaining = max(totalSec - elapsed, 0)
    let progress = totalSec > 0 ? min(Double(elapsed) / Double(totalSec), 1.0) : 0.0

    return SessionProgress(progress: progress, remainingSec: remaining, elapsedSec: elapsed, totalSec: totalSec)
}

/// Calculates break session progress.
public func breakProgress(state: AppState, config: AppConfig, now: Int64) -> SessionProgress {
    let totalSec = config.breakDurationMin * 60
    let referenceNow = state.paused && state.pausedAt > 0 ? state.pausedAt : now
    let elapsed = state.breakStart > 0 ? max(Int(referenceNow - state.breakStart), 0) : 0
    let remaining = max(totalSec - elapsed, 0)
    let progress = totalSec > 0 ? min(Double(elapsed) / Double(totalSec), 1.0) : 0.0

    return SessionProgress(progress: progress, remainingSec: remaining, elapsedSec: elapsed, totalSec: totalSec)
}
