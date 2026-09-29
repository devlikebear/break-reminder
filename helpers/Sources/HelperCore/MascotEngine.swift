import Foundation

public struct Mascot: Equatable {
    public let emoji: String
    public let message: String

    public init(emoji: String, message: String) {
        self.emoji = emoji
        self.message = message
    }
}

/// Selects a mascot (emoji + message) based on the current state.
public func mascotFor(state: AppState, config: AppConfig, now: Int64) -> Mascot {
    if state.paused {
        return Mascot(emoji: "😶", message: L10n.text("Paused for now"))
    }

    if !timerIsRunning(state: state, config: config) {
        return Mascot(emoji: "🌙", message: L10n.text("Your work session is off"))
    }

    if state.mode == "break" {
        let breakElapsed = state.breakStart > 0 ? Int(now - state.breakStart) : 0
        let breakTotal = config.effectiveBreakMin(completedPomodoros: state.pomodoroCount) * 60
        if breakElapsed > breakTotal - 60 {
            return Mascot(emoji: "☕", message: L10n.text("Starting again soon~"))
        }
        return Mascot(emoji: "😴", message: L10n.text("Enjoy your break~ ☕"))
    }

    let workTotal = config.effectiveWorkMin * 60
    let elapsed = state.workSeconds

    // Long continuous work warning (2x or more of the configured work duration)
    if workTotal > 0 && elapsed >= workTotal * 2 {
        return Mascot(emoji: "😰", message: L10n.text("How about a break? 🙏"))
    }

    // Near break time (last 5 minutes)
    if workTotal - elapsed <= 300 && workTotal - elapsed > 0 {
        return Mascot(emoji: "🐹", message: L10n.text("Almost break time~ ☕"))
    }

    return Mascot(emoji: "🐹", message: L10n.text("Focus mode! You can do it 💪"))
}

/// Selects a mascot for achievement moments (e.g., daily goal).
public func mascotForAchievement(dailyWorkMinutes: Int, goalMinutes: Int) -> Mascot? {
    guard goalMinutes > 0, dailyWorkMinutes >= goalMinutes else { return nil }
    return Mascot(emoji: "🎉", message: L10n.text("Great work today! 🏆"))
}
