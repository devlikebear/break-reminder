import XCTest
@testable import HelperCore

final class ConfigParserTests: XCTestCase {
    func testParseFullConfig() {
        let content = """
        work_duration_min: 25
        break_duration_min: 5
        idle_threshold_sec: 60
        check_interval_sec: 30
        voice: "Yuna"
        notifications_enabled: true
        """
        let c = parseConfig(from: content)
        XCTAssertEqual(c.workDurationMin, 25)
        XCTAssertEqual(c.breakDurationMin, 5)
        XCTAssertEqual(c.idleThresholdSec, 60)
        XCTAssertEqual(c.checkIntervalSec, 30)
    }

    func testParseDefaults() {
        let c = parseConfig(from: "")
        XCTAssertEqual(c.workDurationMin, 50)
        XCTAssertEqual(c.breakDurationMin, 10)
        XCTAssertEqual(c.idleThresholdSec, 120)
        XCTAssertEqual(c.checkIntervalSec, 60)
    }

    func testParsePartialConfig() {
        let content = "work_duration_min: 30\n"
        let c = parseConfig(from: content)
        XCTAssertEqual(c.workDurationMin, 30)
        XCTAssertEqual(c.breakDurationMin, 10, "Unset field should use default")
    }

    func testParseIgnoresUnknownKeys() {
        let content = "unknown_key: 999\nwork_duration_min: 25\n"
        let c = parseConfig(from: content)
        XCTAssertEqual(c.workDurationMin, 25)
    }

    func testParseInvalidValues() {
        let content = "work_duration_min: abc\n"
        let c = parseConfig(from: content)
        XCTAssertEqual(c.workDurationMin, 50, "Invalid value should fall back to default")
    }

    func testThemeDefaultAuto() {
        let cfg = AppConfig()
        XCTAssertEqual(cfg.theme, "auto")
    }

    func testThemeParseFromYAML() {
        let yaml = """
        work_duration_min: 50
        theme: dark
        """
        let cfg = parseConfig(from: yaml)
        XCTAssertEqual(cfg.theme, "dark")
    }

    func testThemeParseLight() {
        let cfg = parseConfig(from: "theme: light")
        XCTAssertEqual(cfg.theme, "light")
    }

    func testParseWorkDaysFlowArray() {
        let cfg = parseConfig(from: "work_days: [1, 2, 3, 4, 5]")
        XCTAssertEqual(cfg.workDays, [1, 2, 3, 4, 5])
    }

    func testParseWorkDaysCompact() {
        let cfg = parseConfig(from: "work_days: [1,3,5]")
        XCTAssertEqual(cfg.workDays, [1, 3, 5])
    }

    func testParseScheduleHours() {
        let yaml = """
        work_start_hour: 8
        work_start_minute: 30
        work_end_hour: 17
        work_end_minute: 45
        """
        let cfg = parseConfig(from: yaml)
        XCTAssertEqual(cfg.workStartHour, 8)
        XCTAssertEqual(cfg.workStartMinute, 30)
        XCTAssertEqual(cfg.workEndHour, 17)
        XCTAssertEqual(cfg.workEndMinute, 45)
    }

    func testParseBoolToggles() {
        let yaml = """
        notifications_enabled: false
        tts_enabled: false
        break_activities_enabled: false
        """
        let cfg = parseConfig(from: yaml)
        XCTAssertFalse(cfg.notificationsEnabled)
        XCTAssertFalse(cfg.ttsEnabled)
        XCTAssertFalse(cfg.breakActivitiesEnabled)
    }

    func testParseBreakScreenMode() {
        let cfg = parseConfig(from: "break_screen_mode: block")
        XCTAssertEqual(cfg.breakScreenMode, "block")
    }

    func testParseSessionDetectionKeys() {
        let yaml = """
        auto_session_detect: false
        session_detect_start_hour: 6
        session_detect_end_hour: 23
        session_idle_end_min: 45
        """
        let cfg = parseConfig(from: yaml)
        XCTAssertFalse(cfg.autoSessionDetect)
        XCTAssertEqual(cfg.sessionDetectStartHour, 6)
        XCTAssertEqual(cfg.sessionDetectEndHour, 23)
        XCTAssertEqual(cfg.sessionIdleEndMin, 45)
    }

    func testSessionDetectionDefaults() {
        let cfg = parseConfig(from: "")
        XCTAssertTrue(cfg.autoSessionDetect)
        XCTAssertEqual(cfg.sessionDetectStartHour, 7)
        XCTAssertEqual(cfg.sessionDetectEndHour, 22)
        XCTAssertEqual(cfg.sessionIdleEndMin, 30)
    }

    func testClassicModeUsesClassicDurations() {
        let cfg = parseConfig(from: "")
        XCTAssertFalse(cfg.pomodoroEnabled)
        XCTAssertEqual(cfg.effectiveWorkMin, 50)
        XCTAssertEqual(cfg.effectiveBreakMin(completedPomodoros: 4), 10)
    }

    func testPomodoroModeUsesPomodoroDurations() {
        let yaml = """
        timer_mode: pomodoro
        pomodoro_work_min: 25
        pomodoro_break_min: 5
        pomodoro_long_break_min: 20
        pomodoro_long_break_every: 3
        """
        let cfg = parseConfig(from: yaml)
        XCTAssertTrue(cfg.pomodoroEnabled)
        XCTAssertEqual(cfg.effectiveWorkMin, 25)
        XCTAssertEqual(cfg.effectiveBreakMin(completedPomodoros: 1), 5)
        XCTAssertEqual(cfg.effectiveBreakMin(completedPomodoros: 2), 5)
        XCTAssertEqual(cfg.effectiveBreakMin(completedPomodoros: 3), 20)
        XCTAssertEqual(cfg.effectiveBreakMin(completedPomodoros: 6), 20)
    }

    func testParseNaturalBreakSec() {
        let cfg = parseConfig(from: "natural_break_sec: 600")
        XCTAssertEqual(cfg.naturalBreakSec, 600)
    }
}
