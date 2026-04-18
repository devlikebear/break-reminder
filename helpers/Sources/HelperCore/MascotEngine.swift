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
        return Mascot(emoji: "😶", message: "일시 정지 중이에요")
    }

    if state.mode == "break" {
        let breakElapsed = state.breakStart > 0 ? Int(now - state.breakStart) : 0
        let breakTotal = config.breakDurationMin * 60
        if breakElapsed > breakTotal - 60 {
            return Mascot(emoji: "☕", message: "곧 다시 시작해요~")
        }
        return Mascot(emoji: "😴", message: "푹 쉬고 와요~ ☕")
    }

    let workTotal = config.workDurationMin * 60
    let elapsed = state.workSeconds

    // Long continuous work warning (2x or more of the configured work duration)
    if workTotal > 0 && elapsed >= workTotal * 2 {
        return Mascot(emoji: "😰", message: "쉬어가는 게 어때요? 🙏")
    }

    // Near break time (last 5 minutes)
    if workTotal - elapsed <= 300 && workTotal - elapsed > 0 {
        return Mascot(emoji: "🐹", message: "곧 휴식 시간이에요~ ☕")
    }

    return Mascot(emoji: "🐹", message: "집중 모드! 화이팅 💪")
}

/// Selects a mascot for achievement moments (e.g., daily goal).
public func mascotForAchievement(dailyWorkMinutes: Int, goalMinutes: Int) -> Mascot? {
    guard goalMinutes > 0, dailyWorkMinutes >= goalMinutes else { return nil }
    return Mascot(emoji: "🎉", message: "오늘도 잘 해냈어요! 🏆")
}
