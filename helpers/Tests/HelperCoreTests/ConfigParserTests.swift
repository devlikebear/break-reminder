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
}
