import Foundation

/// Calculates session progress and remaining time.
public struct SessionProgress {
    public let progress: Double   // 0.0 ~ 1.0
    public let remainingSec: Int
    public let elapsedSec: Int
    public let totalSec: Int

    public var remainingFormatted: String { formatTime(remainingSec) }
}

/// Calculates work session progress, interpolating between check intervals.
public func workProgress(state: AppState, config: AppConfig, now: Int64) -> SessionProgress {
    let totalSec = config.workDurationMin * 60
    let sinceLastCheck = state.paused ? 0 : max(Int(now - state.lastCheck), 0)
    let elapsed = state.workSeconds + sinceLastCheck
    let remaining = max(totalSec - elapsed, 0)
    let progress = totalSec > 0 ? min(Double(elapsed) / Double(totalSec), 1.0) : 0.0

    return SessionProgress(progress: progress, remainingSec: remaining, elapsedSec: elapsed, totalSec: totalSec)
}

/// Calculates break session progress.
public func breakProgress(state: AppState, config: AppConfig, now: Int64) -> SessionProgress {
    let totalSec = config.breakDurationMin * 60
    let referenceNow = state.paused && state.pausedAt > 0 ? state.pausedAt : now
    let elapsed = Int(referenceNow - state.breakStart)
    let remaining = max(totalSec - elapsed, 0)
    let progress = totalSec > 0 ? min(Double(elapsed) / Double(totalSec), 1.0) : 0.0

    return SessionProgress(progress: progress, remainingSec: remaining, elapsedSec: elapsed, totalSec: totalSec)
}
