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
}
