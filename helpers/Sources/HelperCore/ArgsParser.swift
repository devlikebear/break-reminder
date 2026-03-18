import Foundation

/// Parsed command-line arguments for break-screen.
public struct BreakScreenArgs {
    public var duration: Int = 600
    public var skipAfter: Int = 120

    public init(duration: Int = 600, skipAfter: Int = 120) {
        self.duration = duration
        self.skipAfter = skipAfter
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
        default:
            break
        }
        i += 1
    }
    return args
}
