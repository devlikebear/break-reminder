import XCTest
@testable import HelperCore

final class MenuBarPresentationTests: XCTestCase {
    func testInterpolatedTodayTotalsIncludeInProgressWork() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_090

        var state = AppState()
        state.mode = "work"
        state.lastCheck = 1_000
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 900
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.checkIntervalSec = 60

        let totals = todayTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 3_690)
        XCTAssertEqual(totals.breakSeconds, 900)
    }

    func testMenuBarPresentationForWorkMode() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_030

        var state = AppState()
        state.mode = "work"
        state.workSeconds = 900
        state.lastCheck = 1_000
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 1_200
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: now)

        XCTAssertEqual(presentation.title, "🐹 31% · 34m left")
        XCTAssertEqual(presentation.statusLine, "Working · 15m elapsed · 34m until break")
        XCTAssertEqual(presentation.statsLine, "Today · Work 1h · Break 20m")
    }

    func testInterpolatedTodayTotalsIncludeInProgressBreakSinceLastCheck() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 2_150

        var state = AppState()
        state.mode = "break"
        state.breakStart = 2_000
        state.lastCheck = 2_100
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 600
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.checkIntervalSec = 60

        let totals = todayTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 7_200)
        XCTAssertEqual(totals.breakSeconds, 650)
    }

    func testMenuBarPresentationForBreakMode() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 2_150

        var state = AppState()
        state.mode = "break"
        state.breakStart = 2_000
        state.lastCheck = 2_100
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 600
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: now)

        XCTAssertEqual(presentation.title, "☕ 25% · 7m left")
        XCTAssertEqual(presentation.statusLine, "On break · 2m elapsed · 7m until work")
        XCTAssertEqual(presentation.statsLine, "Today · Work 2h · Break 10m")
    }

    func testMenuBarPresentationForPausedWorkMode() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_090

        var state = AppState()
        state.mode = "work"
        state.paused = true
        state.workSeconds = 900
        state.lastCheck = 1_000
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 1_200
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: now)

        XCTAssertEqual(presentation.title, "PAUSED (WORK) · 35m left")
        XCTAssertEqual(presentation.statusLine, "PAUSED (WORK) · 15m elapsed · 35m until break")
        XCTAssertEqual(presentation.statsLine, "Today · Work 1h · Break 20m")
    }

    func testMenuBarPresentationForPausedBreakMode() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 2_210

        var state = AppState()
        state.mode = "break"
        state.paused = true
        state.breakStart = 2_000
        state.pausedAt = 2_120
        state.lastCheck = 2_100
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 600
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: now)

        XCTAssertEqual(presentation.title, "PAUSED (BREAK) · 8m left")
        XCTAssertEqual(presentation.statusLine, "PAUSED (BREAK) · 2m elapsed · 8m until work")
        XCTAssertEqual(presentation.statsLine, "Today · Work 2h · Break 10m")
    }

    func testTodayTotalsResetsStalePreviousDayTotals() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        let nowDate = Calendar.current.startOfDay(for: Date()).addingTimeInterval(20)
        let now = Int64(nowDate.timeIntervalSince1970)
        let lastCheck = Int64(nowDate.addingTimeInterval(-30).timeIntervalSince1970)

        var state = AppState()
        state.mode = "work"
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 1_200
        state.lastCheck = lastCheck
        state.lastUpdateDate = formatter.string(from: nowDate.addingTimeInterval(-86_400))

        var config = AppConfig()
        config.checkIntervalSec = 60

        let totals = todayTotals(state: state, config: config, now: now)
        XCTAssertEqual(totals.workSeconds, 20)
        XCTAssertEqual(totals.breakSeconds, 0)
        XCTAssertEqual(totals.date, formatter.string(from: nowDate))
    }
}
