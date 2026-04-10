import XCTest
@testable import HelperCore

final class ProgressCalcTests: XCTestCase {

    // MARK: - Work Progress

    func testWorkProgressAtStart() {
        var state = AppState()
        state.workSeconds = 0
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1000)
        XCTAssertEqual(p.progress, 0.0, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 3000)
        XCTAssertEqual(p.remainingFormatted, "50:00")
    }

    func testWorkProgressInterpolation() {
        var state = AppState()
        state.workSeconds = 900  // 15 min from last check
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 50

        // 30 seconds after last check → effective = 900 + 30 = 930s
        let p = workProgress(state: state, config: config, now: 1030)
        XCTAssertEqual(p.elapsedSec, 930)
        XCTAssertEqual(p.remainingSec, 3000 - 930)
    }

    func testWorkProgressCapsAtOne() {
        var state = AppState()
        state.workSeconds = 3000
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1060)
        XCTAssertEqual(p.progress, 1.0, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 0)
    }

    func testWorkProgressHalfway() {
        var state = AppState()
        state.workSeconds = 1500
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1000)
        XCTAssertEqual(p.progress, 0.5, accuracy: 0.001)
        XCTAssertEqual(p.remainingFormatted, "25:00")
    }

    // MARK: - Break Progress

    func testBreakProgressAtStart() {
        var state = AppState()
        state.breakStart = 1000

        var config = AppConfig()
        config.breakDurationMin = 10

        let p = breakProgress(state: state, config: config, now: 1000)
        XCTAssertEqual(p.progress, 0.0, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 600)
        XCTAssertEqual(p.remainingFormatted, "10:00")
    }

    func testBreakProgressMidway() {
        var state = AppState()
        state.breakStart = 1000

        var config = AppConfig()
        config.breakDurationMin = 10

        // 5 minutes into break
        let p = breakProgress(state: state, config: config, now: 1300)
        XCTAssertEqual(p.progress, 0.5, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 300)
        XCTAssertEqual(p.remainingFormatted, "05:00")
    }

    func testBreakProgressComplete() {
        var state = AppState()
        state.breakStart = 1000

        var config = AppConfig()
        config.breakDurationMin = 10

        let p = breakProgress(state: state, config: config, now: 1700)
        XCTAssertEqual(p.progress, 1.0, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 0)
    }

    func testZeroDurationConfig() {
        var state = AppState()
        state.workSeconds = 0
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 0

        let p = workProgress(state: state, config: config, now: 1000)
        XCTAssertEqual(p.progress, 0.0)
    }

    func testPausedWorkProgressDoesNotKeepAdvancing() {
        var state = AppState()
        state.workSeconds = 900
        state.lastCheck = 1000
        state.paused = true
        state.pausedAt = 1030

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1200)
        XCTAssertEqual(p.elapsedSec, 900)
        XCTAssertEqual(p.remainingSec, 2100)
    }

    func testPausedBreakProgressUsesPauseAnchor() {
        var state = AppState()
        state.breakStart = 1000
        state.paused = true
        state.pausedAt = 1180

        var config = AppConfig()
        config.breakDurationMin = 10

        let p = breakProgress(state: state, config: config, now: 1600)
        XCTAssertEqual(p.elapsedSec, 180)
        XCTAssertEqual(p.remainingSec, 420)
    }
}
