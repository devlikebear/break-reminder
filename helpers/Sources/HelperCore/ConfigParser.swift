import Foundation

/// Represents parsed break-reminder configuration. Mirrors the core set of
/// keys editable via `break-reminder config set` plus a few display values.
public struct AppConfig: Equatable {
    public var workDurationMin: Int = 50
    public var breakDurationMin: Int = 10
    public var idleThresholdSec: Int = 120
    public var naturalBreakSec: Int = 300
    public var checkIntervalSec: Int = 60
    public var workDays: [Int] = [1, 2, 3, 4, 5]
    public var workStartHour: Int = 9
    public var workStartMinute: Int = 0
    public var workEndHour: Int = 18
    public var workEndMinute: Int = 0
    public var notificationsEnabled: Bool = true
    public var ttsEnabled: Bool = true
    public var breakActivitiesEnabled: Bool = true
    public var breakScreenMode: String = "ask"
    public var theme: String = "auto"
    public var autoSessionDetect: Bool = true
    public var sessionDetectStartHour: Int = 7
    public var sessionDetectEndHour: Int = 22
    public var sessionIdleEndMin: Int = 30
    public var timerMode: String = "classic"
    public var pomodoroWorkMin: Int = 25
    public var pomodoroBreakMin: Int = 5
    public var pomodoroLongBreakMin: Int = 15
    public var pomodoroLongBreakEvery: Int = 4

    public init() {}

    /// True when the pomodoro timer replaces the classic long work block.
    public var pomodoroEnabled: Bool { timerMode == "pomodoro" }

    /// Work block length of the active timer mode, in minutes.
    public var effectiveWorkMin: Int { pomodoroEnabled ? pomodoroWorkMin : workDurationMin }

    /// Break length that follows the given number of completed pomodoros.
    public func effectiveBreakMin(completedPomodoros: Int) -> Int {
        guard pomodoroEnabled else { return breakDurationMin }
        if pomodoroLongBreakEvery > 0,
           completedPomodoros > 0,
           completedPomodoros % pomodoroLongBreakEvery == 0 {
            return pomodoroLongBreakMin
        }
        return pomodoroBreakMin
    }
}

/// Parses configuration from simple YAML content (flat key: value pairs and
/// flow-style integer arrays such as `work_days: [1, 2, 3, 4, 5]`).
public func parseConfig(from content: String) -> AppConfig {
    var c = AppConfig()
    for line in content.components(separatedBy: "\n") {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        if trimmed.isEmpty || trimmed.hasPrefix("#") { continue }
        let parts = trimmed.split(separator: ":", maxSplits: 1)
        guard parts.count == 2 else { continue }
        let key = String(parts[0]).trimmingCharacters(in: .whitespaces)
        let rawVal = String(parts[1]).trimmingCharacters(in: .whitespaces)
        let val = stripQuotes(rawVal)
        switch key {
        case "work_duration_min":        c.workDurationMin = Int(val) ?? c.workDurationMin
        case "break_duration_min":       c.breakDurationMin = Int(val) ?? c.breakDurationMin
        case "idle_threshold_sec":       c.idleThresholdSec = Int(val) ?? c.idleThresholdSec
        case "natural_break_sec":        c.naturalBreakSec = Int(val) ?? c.naturalBreakSec
        case "check_interval_sec":       c.checkIntervalSec = Int(val) ?? c.checkIntervalSec
        case "work_start_hour":          c.workStartHour = Int(val) ?? c.workStartHour
        case "work_start_minute":        c.workStartMinute = Int(val) ?? c.workStartMinute
        case "work_end_hour":            c.workEndHour = Int(val) ?? c.workEndHour
        case "work_end_minute":          c.workEndMinute = Int(val) ?? c.workEndMinute
        case "notifications_enabled":    c.notificationsEnabled = (val == "true")
        case "tts_enabled":              c.ttsEnabled = (val == "true")
        case "break_activities_enabled": c.breakActivitiesEnabled = (val == "true")
        case "break_screen_mode":        c.breakScreenMode = val
        case "theme":                    c.theme = val
        case "auto_session_detect":      c.autoSessionDetect = (val == "true")
        case "session_detect_start_hour": c.sessionDetectStartHour = Int(val) ?? c.sessionDetectStartHour
        case "session_detect_end_hour":  c.sessionDetectEndHour = Int(val) ?? c.sessionDetectEndHour
        case "session_idle_end_min":     c.sessionIdleEndMin = Int(val) ?? c.sessionIdleEndMin
        case "timer_mode":               c.timerMode = val
        case "pomodoro_work_min":        c.pomodoroWorkMin = Int(val) ?? c.pomodoroWorkMin
        case "pomodoro_break_min":       c.pomodoroBreakMin = Int(val) ?? c.pomodoroBreakMin
        case "pomodoro_long_break_min":  c.pomodoroLongBreakMin = Int(val) ?? c.pomodoroLongBreakMin
        case "pomodoro_long_break_every": c.pomodoroLongBreakEvery = Int(val) ?? c.pomodoroLongBreakEvery
        case "work_days":
            let stripped = val.trimmingCharacters(in: CharacterSet(charactersIn: "[]"))
            let parsed = stripped.split(separator: ",").compactMap {
                Int($0.trimmingCharacters(in: .whitespaces))
            }
            if !parsed.isEmpty {
                c.workDays = parsed
            }
        default: break
        }
    }
    return c
}

private func stripQuotes(_ s: String) -> String {
    var out = s
    if (out.hasPrefix("\"") && out.hasSuffix("\"")) ||
        (out.hasPrefix("'") && out.hasSuffix("'")) {
        out.removeFirst()
        out.removeLast()
    }
    return out
}
