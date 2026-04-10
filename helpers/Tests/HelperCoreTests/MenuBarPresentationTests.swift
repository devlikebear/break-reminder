import XCTest
@testable import HelperCore

final class MenuBarPresentationTests: XCTestCase {
    func testInterpolatedTodayTotalsIncludeInProgressWork() {
        var state = AppState()
        state.mode = "work"
        state.lastCheck = 1_000
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 900

        var config = AppConfig()
        config.checkIntervalSec = 60

        let totals = todayTotals(state: state, config: config, now: 1_090)
        XCTAssertEqual(totals.workSeconds, 3_690)
        XCTAssertEqual(totals.breakSeconds, 900)
    }

    func testMenuBarPresentationForWorkMode() {
        var state = AppState()
        state.mode = "work"
        state.workSeconds = 900
        state.lastCheck = 1_000
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 1_200

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: 1_030)

        XCTAssertEqual(presentation.title, "🟢 31% · 34m left")
        XCTAssertEqual(presentation.statusLine, "Working · 15m elapsed · 34m until break")
        XCTAssertEqual(presentation.statsLine, "Today · Work 1h · Break 20m")
    }

    func testInterpolatedTodayTotalsIncludeInProgressBreakSinceLastCheck() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = 2_000
        state.lastCheck = 2_100
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 600

        var config = AppConfig()
        config.checkIntervalSec = 60

        let totals = todayTotals(state: state, config: config, now: 2_150)
        XCTAssertEqual(totals.workSeconds, 7_200)
        XCTAssertEqual(totals.breakSeconds, 650)
    }

    func testMenuBarPresentationForBreakMode() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = 2_000
        state.lastCheck = 2_100
        state.todayWorkSeconds = 7_200
        state.todayBreakSeconds = 600

        var config = AppConfig()
        config.workDurationMin = 50
        config.breakDurationMin = 10
        config.checkIntervalSec = 60

        let presentation = menuBarPresentation(state: state, config: config, now: 2_150)

        XCTAssertEqual(presentation.title, "☕ 25% · 7m left")
        XCTAssertEqual(presentation.statusLine, "On break · 2m elapsed · 7m until work")
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
