import XCTest
@testable import HelperCore

final class TimeFormatterTests: XCTestCase {
    func testZero() {
        XCTAssertEqual(formatTime(0), "00:00")
    }

    func testSeconds() {
        XCTAssertEqual(formatTime(45), "00:45")
    }

    func testMinutesAndSeconds() {
        XCTAssertEqual(formatTime(125), "02:05")
    }

    func testExactMinutes() {
        XCTAssertEqual(formatTime(600), "10:00")
    }

    func testLargeValue() {
        XCTAssertEqual(formatTime(3661), "61:01")
    }
}
