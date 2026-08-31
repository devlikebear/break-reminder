import Foundation

/// Represents the break-reminder application state.
public struct AppState: Equatable {
    public var workSeconds: Int = 0
    public var mode: String = "work"
    public var lastCheck: Int64 = 0
    public var breakStart: Int64 = 0
    public var snoozeUntil: Int64 = 0
    public var paused: Bool = false
    public var pausedAt: Int64 = 0
    public var pauseReason: String = ""
    public var pauseUntil: Int64 = 0
    public var todayWorkSeconds: Int = 0
    public var todayBreakSeconds: Int = 0
    public var lastUpdateDate: String = ""
    public var sessionState: String = ""
    public var sessionStart: Int64 = 0
    public var sessionEnd: Int64 = 0
    public var sessionManual: Bool = false
    public var lastActivity: Int64 = 0
    public var pomodoroCount: Int = 0
    public var todayPomodoros: Int = 0

    public init() {}

    /// True while a work session is running and the Go timer is ticking.
    public var isSessionActive: Bool { sessionState == "active" }
}

/// Parses state from key=value formatted string content.
public func parseState(from content: String) -> AppState {
    var s = AppState()
    for line in content.components(separatedBy: "\n") {
        let parts = line.split(separator: "=", maxSplits: 1)
        guard parts.count == 2 else { continue }
        let key = String(parts[0])
        let val = String(parts[1])
        switch key {
        case "WORK_SECONDS":        s.workSeconds = Int(val) ?? 0
        case "MODE":                s.mode = val
        case "LAST_CHECK":          s.lastCheck = Int64(val) ?? 0
        case "BREAK_START":         s.breakStart = Int64(val) ?? 0
        case "SNOOZE_UNTIL":        s.snoozeUntil = Int64(val) ?? 0
        case "PAUSED":              s.paused = (val == "true")
        case "PAUSED_AT":           s.pausedAt = Int64(val) ?? 0
        case "PAUSE_REASON":
            if val == "meeting" || val == "focus" || val == "afk" {
                s.pauseReason = val
            } else {
                s.pauseReason = ""
            }
        case "PAUSE_UNTIL":         s.pauseUntil = Int64(val) ?? 0
        case "TODAY_WORK_SECONDS":  s.todayWorkSeconds = Int(val) ?? 0
        case "TODAY_BREAK_SECONDS": s.todayBreakSeconds = Int(val) ?? 0
        case "LAST_UPDATE_DATE":    s.lastUpdateDate = val
        case "SESSION_STATE":
            if val == "active" || val == "ended" {
                s.sessionState = val
            } else {
                s.sessionState = ""
            }
        case "SESSION_START":       s.sessionStart = Int64(val) ?? 0
        case "SESSION_END":         s.sessionEnd = Int64(val) ?? 0
        case "SESSION_MANUAL":      s.sessionManual = (val == "true")
        case "LAST_ACTIVITY":       s.lastActivity = Int64(val) ?? 0
        case "POMODORO_COUNT":      s.pomodoroCount = Int(val) ?? 0
        case "TODAY_POMODOROS":     s.todayPomodoros = Int(val) ?? 0
        default: break
        }
    }
    return s
}

/// Serializes state to key=value format.
public func serializeState(_ s: AppState) -> String {
    return [
        "WORK_SECONDS=\(s.workSeconds)",
        "MODE=\(s.mode)",
        "LAST_CHECK=\(s.lastCheck)",
        "BREAK_START=\(s.breakStart)",
        "SNOOZE_UNTIL=\(s.snoozeUntil)",
        "PAUSED=\(s.paused ? "true" : "false")",
        "PAUSED_AT=\(s.pausedAt)",
        "PAUSE_REASON=\(s.pauseReason)",
        "PAUSE_UNTIL=\(s.pauseUntil)",
        "TODAY_WORK_SECONDS=\(s.todayWorkSeconds)",
        "TODAY_BREAK_SECONDS=\(s.todayBreakSeconds)",
        "LAST_UPDATE_DATE=\(s.lastUpdateDate)",
        "SESSION_STATE=\(s.sessionState)",
        "SESSION_START=\(s.sessionStart)",
        "SESSION_END=\(s.sessionEnd)",
        "SESSION_MANUAL=\(s.sessionManual ? "true" : "false")",
        "LAST_ACTIVITY=\(s.lastActivity)",
        "POMODORO_COUNT=\(s.pomodoroCount)",
        "TODAY_POMODOROS=\(s.todayPomodoros)",
    ].joined(separator: "\n")
}
