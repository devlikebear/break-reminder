import XCTest
@testable import HelperCore

final class StateParserTests: XCTestCase {
    func testParseValidState() {
        let content = """
        WORK_SECONDS=900
        MODE=work
        LAST_CHECK=1710000000
        BREAK_START=0
        SNOOZE_UNTIL=1710000300
        PAUSED=true
        PAUSED_AT=1710000200
        TODAY_WORK_SECONDS=3600
        TODAY_BREAK_SECONDS=600
        LAST_UPDATE_DATE=2026-03-18
        """
        let s = parseState(from: content)
        XCTAssertEqual(s.workSeconds, 900)
        XCTAssertEqual(s.mode, "work")
        XCTAssertEqual(s.lastCheck, 1710000000)
        XCTAssertEqual(s.breakStart, 0)
        XCTAssertEqual(s.snoozeUntil, 1710000300)
        XCTAssertTrue(s.paused)
        XCTAssertEqual(s.pausedAt, 1710000200)
        XCTAssertEqual(s.todayWorkSeconds, 3600)
        XCTAssertEqual(s.todayBreakSeconds, 600)
        XCTAssertEqual(s.lastUpdateDate, "2026-03-18")
    }

    func testParseBreakMode() {
        let content = "MODE=break\nBREAK_START=1710000000\n"
        let s = parseState(from: content)
        XCTAssertEqual(s.mode, "break")
        XCTAssertEqual(s.breakStart, 1710000000)
    }

    func testParseEmptyContent() {
        let s = parseState(from: "")
        XCTAssertEqual(s.workSeconds, 0)
        XCTAssertEqual(s.mode, "work")
    }

    func testParseInvalidLines() {
        let content = "GARBAGE\nNO_EQUALS\nWORK_SECONDS=100\n"
        let s = parseState(from: content)
        XCTAssertEqual(s.workSeconds, 100)
    }

    func testParseInvalidNumbers() {
        let content = "WORK_SECONDS=abc\nLAST_CHECK=xyz\n"
        let s = parseState(from: content)
        XCTAssertEqual(s.workSeconds, 0)
        XCTAssertEqual(s.lastCheck, 0)
    }

    func testSerializeRoundtrip() {
        var original = AppState()
        original.workSeconds = 1500
        original.mode = "break"
        original.lastCheck = 1710000000
        original.breakStart = 1709999000
        original.snoozeUntil = 1710000300
        original.paused = true
        original.pausedAt = 1710000200
        original.todayWorkSeconds = 7200
        original.todayBreakSeconds = 1200
        original.lastUpdateDate = "2026-03-18"

        let serialized = serializeState(original)
        let parsed = parseState(from: serialized)

        XCTAssertEqual(parsed, original)
    }

    func testParsePausedFields() {
        let content = "PAUSED=true\nPAUSED_AT=1710000300\n"
        let s = parseState(from: content)
        XCTAssertTrue(s.paused)
        XCTAssertEqual(s.pausedAt, 1710000300)
    }
}
