import Foundation

public enum TimerPhase: Equatable {
    case idle, working, onBreak, paused
}

/// Everything the popover and status item draw for one engine status.
public struct PopoverPresentation: Equatable {
    public var phase: TimerPhase
    public var menuTitle: String
    public var statusLine: String
    public var statsLine: String
    public var pomodoroLine: String
    public var progress: Double
    public var sessionActive: Bool
    public var canSnooze: Bool
    public var canSwitchMode: Bool
}

/// Builds the popover view state, reusing the menu bar presentation helpers
/// so the app reads exactly like the legacy menu bar.
public func popoverPresentation(status: EngineStatus, now: Int64) -> PopoverPresentation {
    let state = status.appState
    let config = status.appConfig
    let base = menuBarPresentation(state: state, config: config, now: now)
    let running = timerIsRunning(state: state, config: config)

    let phase: TimerPhase
    if state.paused {
        phase = .paused
    } else if !running {
        phase = .idle
    } else {
        phase = state.mode == "break" ? .onBreak : .working
    }

    var progress = 0.0
    if phase != .idle {
        progress = state.mode == "break"
            ? breakProgress(state: state, config: config, now: now).progress
            : workProgress(state: state, config: config, now: now).progress
    }

    return PopoverPresentation(
        phase: phase,
        menuTitle: base.title,
        statusLine: base.statusLine,
        statsLine: base.statsLine,
        pomodoroLine: pomodoroStatus(state: state, config: config),
        progress: progress,
        sessionActive: state.isSessionActive,
        canSnooze: phase == .onBreak,
        canSwitchMode: !state.paused && state.mode == "work"
    )
}
