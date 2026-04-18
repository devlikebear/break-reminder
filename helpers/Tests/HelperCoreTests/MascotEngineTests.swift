import XCTest
@testable import HelperCore

final class MascotEngineTests: XCTestCase {
    func testWorkingStateReturnsHamster() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 300 // 5 min
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "🐹")
        XCTAssertFalse(mascot.message.isEmpty)
    }

    func testBreakStateReturnsSleeping() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = Int64(Date().timeIntervalSince1970) - 60
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😴")
    }

    func testLongWorkReturnsConcerned() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 7200 // 2 hours
        var config = AppConfig()
        config.workDurationMin = 50 // break every 50 min; 2 hours = 2.4x threshold
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😰")
    }

    func testPausedReturnsNeutral() {
        var state = AppState()
        state.mode = "work"
        state.paused = true
        let config = AppConfig()
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "😶")
    }

    func testNearBreakReturnsHamsterWithBreakMessage() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 50 * 60 - 120 // 2 min remaining to break
        var config = AppConfig()
        config.workDurationMin = 50
        let mascot = mascotFor(state: state, config: config, now: Int64(Date().timeIntervalSince1970))

        XCTAssertEqual(mascot.emoji, "🐹")
        XCTAssertTrue(mascot.message.contains("휴식"), "message should hint at break: \(mascot.message)")
    }

    func testAchievementMascotBelowGoal() {
        let result = mascotForAchievement(dailyWorkMinutes: 100, goalMinutes: 240)
        XCTAssertNil(result)
    }

    func testAchievementMascotReachedGoal() {
        let result = mascotForAchievement(dailyWorkMinutes: 240, goalMinutes: 240)
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.emoji, "🎉")
    }
}
