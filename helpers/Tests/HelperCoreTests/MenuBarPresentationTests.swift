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
        state.sessionState = "active"
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

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "31% · 34분 남음" : "31% · 34m left")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "업무 중 · 15분 경과 · 휴식까지 34분" : "Working · 15m elapsed · 34m until break")
        XCTAssertEqual(presentation.statsLine, AppLanguage.current == .korean ? "오늘: 작업 1시간 · 휴식 20분" : "Today · Work 1h · Break 20m")
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
        state.sessionState = "active"
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

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "25% · 7분 남음" : "25% · 7m left")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "휴식 중 · 2분 경과 · 업무까지 7분" : "On break · 2m elapsed · 7m until work")
        XCTAssertEqual(presentation.statsLine, AppLanguage.current == .korean ? "오늘: 작업 2시간 · 휴식 10분" : "Today · Work 2h · Break 10m")
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

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "일시정지 (업무) · 35분 남음" : "PAUSED (WORK) · 35m left")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "일시정지 (업무) · 15분 경과 · 휴식까지 35분" : "PAUSED (WORK) · 15m elapsed · 35m until break")
        XCTAssertEqual(presentation.statsLine, AppLanguage.current == .korean ? "오늘: 작업 1시간 · 휴식 20분" : "Today · Work 1h · Break 20m")
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

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "일시정지 (휴식) · 8분 남음" : "PAUSED (BREAK) · 8m left")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "일시정지 (휴식) · 2분 경과 · 업무까지 8분" : "PAUSED (BREAK) · 2m elapsed · 8m until work")
        XCTAssertEqual(presentation.statsLine, AppLanguage.current == .korean ? "오늘: 작업 2시간 · 휴식 10분" : "Today · Work 2h · Break 10m")
    }

    func testMenuBarPresentationForStoppedSession() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_030

        var state = AppState()
        state.mode = "work"
        state.sessionState = "ended"
        state.sessionManual = true
        state.todayWorkSeconds = 3_600
        state.todayBreakSeconds = 1_200
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        let presentation = menuBarPresentation(state: state, config: AppConfig(), now: now)

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "꺼짐" : "Off")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "수동 종료됨 · 세션을 시작하면 재개됩니다" : "Stopped by hand · start a session to resume")
        XCTAssertEqual(presentation.statsLine, AppLanguage.current == .korean ? "오늘: 작업 1시간 · 휴식 20분" : "Today · Work 1h · Break 20m")
    }

    func testMenuBarPresentationBeforeFirstSessionOfTheDay() {
        let presentation = menuBarPresentation(state: AppState(), config: AppConfig(), now: 1_030)

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "대기" : "Idle")
        XCTAssertEqual(presentation.statusLine, AppLanguage.current == .korean ? "대기 중 · 첫 활동 시 시작됩니다" : "Waiting · starts on your first activity")
    }

    func testMenuBarPresentationPomodoroUsesPomodoroDurations() {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let now: Int64 = 1_000

        var state = AppState()
        state.sessionState = "active"
        state.mode = "work"
        state.workSeconds = 300
        state.lastCheck = 1_000
        state.lastUpdateDate = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(now)))

        var config = AppConfig()
        config.timerMode = "pomodoro"

        let presentation = menuBarPresentation(state: state, config: config, now: now)

        XCTAssertEqual(presentation.title, AppLanguage.current == .korean ? "20% · 20분 남음" : "20% · 20m left")
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
