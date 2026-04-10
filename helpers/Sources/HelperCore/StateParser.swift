import Foundation

/// Represents the break-reminder application state.
public struct AppState: Equatable {
    public var workSeconds: Int = 0
    public var mode: String = "work"
    public var lastCheck: Int64 = 0
    public var breakStart: Int64 = 0
    public var paused: Bool = false
    public var pausedAt: Int64 = 0
    public var todayWorkSeconds: Int = 0
    public var todayBreakSeconds: Int = 0
    public var lastUpdateDate: String = ""

    public init() {}
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
        case "PAUSED":              s.paused = (val == "true")
        case "PAUSED_AT":           s.pausedAt = Int64(val) ?? 0
        case "TODAY_WORK_SECONDS":  s.todayWorkSeconds = Int(val) ?? 0
        case "TODAY_BREAK_SECONDS": s.todayBreakSeconds = Int(val) ?? 0
        case "LAST_UPDATE_DATE":    s.lastUpdateDate = val
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
        "PAUSED=\(s.paused ? "true" : "false")",
        "PAUSED_AT=\(s.pausedAt)",
        "TODAY_WORK_SECONDS=\(s.todayWorkSeconds)",
        "TODAY_BREAK_SECONDS=\(s.todayBreakSeconds)",
        "LAST_UPDATE_DATE=\(s.lastUpdateDate)",
    ].joined(separator: "\n")
}
