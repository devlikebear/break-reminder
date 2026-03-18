import Foundation

/// Represents parsed break-reminder configuration.
public struct AppConfig: Equatable {
    public var workDurationMin: Int = 50
    public var breakDurationMin: Int = 10
    public var idleThresholdSec: Int = 120
    public var checkIntervalSec: Int = 60

    public init() {}
}

/// Parses configuration from simple YAML content (flat key: value only).
public func parseConfig(from content: String) -> AppConfig {
    var c = AppConfig()
    for line in content.components(separatedBy: "\n") {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        let parts = trimmed.split(separator: ":", maxSplits: 1)
        guard parts.count == 2 else { continue }
        let key = String(parts[0]).trimmingCharacters(in: .whitespaces)
        let val = String(parts[1]).trimmingCharacters(in: .whitespaces)
        switch key {
        case "work_duration_min":  c.workDurationMin = Int(val) ?? 50
        case "break_duration_min": c.breakDurationMin = Int(val) ?? 10
        case "idle_threshold_sec": c.idleThresholdSec = Int(val) ?? 120
        case "check_interval_sec": c.checkIntervalSec = Int(val) ?? 60
        default: break
        }
    }
    return c
}
