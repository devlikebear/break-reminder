public enum GuidedBreakPhase: Equatable {
    case ready
    case running(remainingSeconds: Int)
    case completed(remainingDisplaySeconds: Int)
}

public enum GuidedBreakTickResult: Equatable {
    case stay
    case phaseChanged
    case dismiss
}

public struct GuidedBreakSession {
    public static let activityID = "standing-neck-shoulder-stretch-v1"
    public static let activityDurationSeconds = 120
    public static let completionDisplaySeconds = 3

    public private(set) var phase: GuidedBreakPhase = .ready

    public init() {}

    public mutating func start(availableBreakSeconds: Int) -> Bool {
        guard case .ready = phase,
              availableBreakSeconds >= Self.activityDurationSeconds + Self.completionDisplaySeconds else {
            return false
        }
        phase = .running(remainingSeconds: Self.activityDurationSeconds)
        return true
    }

    public mutating func cancel() {
        guard case .running = phase else { return }
        phase = .ready
    }

    public mutating func tick() -> GuidedBreakTickResult {
        switch phase {
        case .ready:
            return .stay
        case let .running(remainingSeconds) where remainingSeconds > 1:
            phase = .running(remainingSeconds: remainingSeconds - 1)
            return .stay
        case .running:
            phase = .completed(remainingDisplaySeconds: Self.completionDisplaySeconds)
            return .phaseChanged
        case let .completed(remainingDisplaySeconds) where remainingDisplaySeconds > 1:
            phase = .completed(remainingDisplaySeconds: remainingDisplaySeconds - 1)
            return .stay
        case .completed:
            return .dismiss
        }
    }

    public func instructionText() -> String {
        switch phase {
        case .ready:
            return ""
        case let .running(remainingSeconds) where remainingSeconds >= 81:
            return L10n.text("Stand comfortably and relax your shoulders.")
        case let .running(remainingSeconds) where remainingSeconds >= 41:
            return L10n.text("Slowly tilt your head from side to side. Stop if it hurts.")
        case .running:
            return L10n.text("Slowly roll your shoulders back and breathe.")
        case .completed:
            return L10n.text("All done — relax and enjoy the rest of your break.")
        }
    }
}
