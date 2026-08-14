import Foundation

public enum MenuBarAnimationFrameCounts {
    public static let running = 6
    public static let resting = 2
}

public enum MenuBarAnimationKind: Equatable {
    case running
    case resting
    case sleeping
}

public struct MenuBarAnimationFrame: Equatable {
    public let kind: MenuBarAnimationKind
    public let frameIndex: Int

    public init(kind: MenuBarAnimationKind, frameIndex: Int) {
        self.kind = kind
        self.frameIndex = frameIndex
    }
}

/// Selects the menu-bar mascot frame for the current timer state.
/// The caller advances `tick` at a steady cadence (250 ms in MenuBarApp).
public func menuBarAnimation(state: AppState, tick: Int) -> MenuBarAnimationFrame {
    func wrapped(_ value: Int, modulo: Int) -> Int {
        ((value % modulo) + modulo) % modulo
    }

    if state.paused {
        return MenuBarAnimationFrame(kind: .sleeping, frameIndex: 0)
    }
    if state.mode == "break" {
        // Hold each breathing pose for two ticks to create a calm 2 fps cycle.
        return MenuBarAnimationFrame(
            kind: .resting,
            frameIndex: wrapped(tick / 2, modulo: MenuBarAnimationFrameCounts.resting)
        )
    }
    return MenuBarAnimationFrame(
        kind: .running,
        frameIndex: wrapped(tick, modulo: MenuBarAnimationFrameCounts.running)
    )
}
