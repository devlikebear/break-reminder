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
        state.workSeconds = 900
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1030)
        XCTAssertEqual(p.elapsedSec, 930)
        XCTAssertEqual(p.remainingSec, 3000 - 930)
    }

    func testWorkProgressDoesNotInterpolateFromZeroLastCheck() {
        var state = AppState()
        state.workSeconds = 0
        state.lastCheck = 0

        var config = AppConfig()
        config.workDurationMin = 50

        let p = workProgress(state: state, config: config, now: 1_710_000_000)
        XCTAssertEqual(p.elapsedSec, 0)
        XCTAssertEqual(p.remainingSec, 3000)
        XCTAssertEqual(p.progress, 0.0, accuracy: 0.001)
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

        let p = breakProgress(state: state, config: config, now: 1300)
        XCTAssertEqual(p.progress, 0.5, accuracy: 0.001)
        XCTAssertEqual(p.remainingSec, 300)
        XCTAssertEqual(p.remainingFormatted, "05:00")
    }

    func testBreakProgressDoesNotInterpolateFromZeroBreakStart() {
        var state = AppState()
        state.breakStart = 0

        var config = AppConfig()
        config.breakDurationMin = 10

        let p = breakProgress(state: state, config: config, now: 1_710_000_000)
        XCTAssertEqual(p.elapsedSec, 0)
        XCTAssertEqual(p.remainingSec, 600)
        XCTAssertEqual(p.progress, 0.0, accuracy: 0.001)
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

    func testZeroDurationConfig() {
        var state = AppState()
        state.workSeconds = 0
        state.lastCheck = 1000

        var config = AppConfig()
        config.workDurationMin = 0

        let p = workProgress(state: state, config: config, now: 1000)
        XCTAssertEqual(p.progress, 0.0)
    }

    func testLiveTodayWorkSecondsSkipsZeroLastCheck() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_710_000_000

        var config = AppConfig()
        config.checkIntervalSec = 60

        var state = AppState()
        state.mode = "work"
        state.todayWorkSeconds = 120
        state.lastCheck = 0
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        XCTAssertEqual(liveTodayWorkSeconds(state: state, config: config, now: now), 120)
    }

    func testLiveTodayBreakSecondsInterpolatesCurrentBreak() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_710_000_090

        var config = AppConfig()
        config.checkIntervalSec = 60

        var state = AppState()
        state.mode = "break"
        state.todayBreakSeconds = 600
        state.lastCheck = 1_710_000_000
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        XCTAssertEqual(liveTodayBreakSeconds(state: state, config: config, now: now), 690)
    }

    func testLiveTodayTotalsSkipsLongGapsToMatchTimerSemantics() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now = Int64(Date().timeIntervalSince1970)

        var config = AppConfig()
        config.checkIntervalSec = 60

        var state = AppState()
        state.mode = "work"
        state.todayWorkSeconds = 120
        state.lastCheck = now - 900
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        let totals = liveDailyTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 120)
        XCTAssertEqual(totals.breakSeconds, 0)
    }

    func testLiveTodayTotalsResetsStalePreviousDayWorkTotals() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        let nowDate = Calendar.current.startOfDay(for: Date()).addingTimeInterval(20)
        let now = Int64(nowDate.timeIntervalSince1970)
        let lastCheck = Int64(nowDate.addingTimeInterval(-30).timeIntervalSince1970)
        let yesterday = formatter.string(from: nowDate.addingTimeInterval(-86_400))
        let today = formatter.string(from: nowDate)

        var config = AppConfig()
        config.checkIntervalSec = 60

        var state = AppState()
        state.mode = "work"
        state.todayWorkSeconds = 7200
        state.todayBreakSeconds = 1800
        state.lastCheck = lastCheck
        state.lastUpdateDate = yesterday

        let totals = liveDailyTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 20)
        XCTAssertEqual(totals.breakSeconds, 0)
        XCTAssertEqual(totals.date, today)
    }

    func testLiveTodayTotalsCarriesForwardCurrentBreakAfterMidnight() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        let nowDate = Calendar.current.startOfDay(for: Date()).addingTimeInterval(20)
        let now = Int64(nowDate.timeIntervalSince1970)
        let lastCheck = Int64(nowDate.addingTimeInterval(-30).timeIntervalSince1970)
        let yesterday = formatter.string(from: nowDate.addingTimeInterval(-86_400))
        let today = formatter.string(from: nowDate)

        var config = AppConfig()
        config.checkIntervalSec = 60

        var state = AppState()
        state.mode = "break"
        state.todayWorkSeconds = 7200
        state.todayBreakSeconds = 1200
        state.lastCheck = lastCheck
        state.lastUpdateDate = yesterday

        let totals = liveDailyTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 0)
        XCTAssertEqual(totals.breakSeconds, 20)
        XCTAssertEqual(totals.date, today)
    }
}
