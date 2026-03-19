import Foundation

/// Parsed command-line arguments for break-screen.
public struct BreakScreenArgs {
    public var duration: Int = 600
    public var skipAfter: Int = 120
    public var todayWorkMin: Int = 0
    public var todayBreakMin: Int = 0

    public init(duration: Int = 600, skipAfter: Int = 120, todayWorkMin: Int = 0, todayBreakMin: Int = 0) {
        self.duration = duration
        self.skipAfter = skipAfter
        self.todayWorkMin = todayWorkMin
        self.todayBreakMin = todayBreakMin
    }
}

/// Parses break-screen CLI arguments from the given array.
/// The first element should be the program name (skipped).
public func parseBreakScreenArgs(_ argv: [String]) -> BreakScreenArgs {
    var args = BreakScreenArgs()
    var i = 1
    while i < argv.count {
        switch argv[i] {
        case "--duration":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.duration = v }
        case "--skip-after":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.skipAfter = v }
        case "--work-min":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.todayWorkMin = v }
        case "--break-min":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.todayBreakMin = v }
        default:
            break
        }
        i += 1
    }
    return args
}

/// Formats minutes as "Xh Ym" or "Ym" for display.
public func formatMinutes(_ min: Int) -> String {
    if min >= 60 {
        let h = min / 60
        let m = min % 60
        return m > 0 ? "\(h)h \(m)m" : "\(h)h"
    }
    return "\(min)m"
}
