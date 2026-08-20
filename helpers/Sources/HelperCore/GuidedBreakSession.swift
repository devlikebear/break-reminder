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
            return "편안히 서서 어깨의 힘을 빼세요."
        case let .running(remainingSeconds) where remainingSeconds >= 41:
            return "고개를 천천히 좌우로 기울이고, 통증이 있으면 멈추세요."
        case .running:
            return "어깨를 뒤로 천천히 돌리며 호흡하세요."
        case .completed:
            return "완료했어요 — 편안하게 남은 휴식을 이어가세요."
        }
    }
}
