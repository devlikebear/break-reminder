import XCTest
@testable import HelperCore

final class ArgsParserTests: XCTestCase {
    func testDefaults() {
        let args = parseBreakScreenArgs(["break-screen"])
        XCTAssertEqual(args.duration, 600)
        XCTAssertEqual(args.skipAfter, 120)
    }

    func testDuration() {
        let args = parseBreakScreenArgs(["break-screen", "--duration", "300"])
        XCTAssertEqual(args.duration, 300)
        XCTAssertEqual(args.skipAfter, 120)
    }

    func testSkipAfter() {
        let args = parseBreakScreenArgs(["break-screen", "--skip-after", "60"])
        XCTAssertEqual(args.duration, 600)
        XCTAssertEqual(args.skipAfter, 60)
    }

    func testBothArgs() {
        let args = parseBreakScreenArgs(["break-screen", "--duration", "30", "--skip-after", "5"])
        XCTAssertEqual(args.duration, 30)
        XCTAssertEqual(args.skipAfter, 5)
    }

    func testInvalidNumber() {
        let args = parseBreakScreenArgs(["break-screen", "--duration", "abc"])
        XCTAssertEqual(args.duration, 600, "Should keep default for invalid number")
    }

    func testEmptyArgs() {
        let args = parseBreakScreenArgs([])
        XCTAssertEqual(args.duration, 600)
    }

    func testMissingValue() {
        let args = parseBreakScreenArgs(["break-screen", "--duration"])
        XCTAssertEqual(args.duration, 600, "Should keep default when value is missing")
    }

    func testWorkBreakMinArgs() {
        let args = parseBreakScreenArgs(["break-screen", "--duration", "300", "--work-min", "125", "--break-min", "30"])
        XCTAssertEqual(args.todayWorkMin, 125)
        XCTAssertEqual(args.todayBreakMin, 30)
    }

    func testWorkBreakMinDefaults() {
        let args = parseBreakScreenArgs(["break-screen"])
        XCTAssertEqual(args.todayWorkMin, 0)
        XCTAssertEqual(args.todayBreakMin, 0)
    }

    func testFormatMinutesUnderHour() {
        XCTAssertEqual(formatMinutes(0), AppLanguage.current == .korean ? "0분" : "0m")
        XCTAssertEqual(formatMinutes(45), AppLanguage.current == .korean ? "45분" : "45m")
    }

    func testFormatMinutesOverHour() {
        XCTAssertEqual(formatMinutes(60), AppLanguage.current == .korean ? "1시간" : "1h")
        XCTAssertEqual(formatMinutes(125), AppLanguage.current == .korean ? "2시간 5분" : "2h 5m")
        XCTAssertEqual(formatMinutes(180), AppLanguage.current == .korean ? "3시간" : "3h")
    }
}
