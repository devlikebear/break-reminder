import XCTest
@testable import HelperCore

final class HistoryParserTests: XCTestCase {
    func testParseFullEntry() {
        let json = """
        [{
          "date": "2026-04-17",
          "work_min": 280,
          "break_min": 60,
          "sessions": 7,
          "activities": 3,
          "hourly_work": [0,0,0,0,0,0,0,0,0,45,55,50,10,40,50,35,20,0,0,0,0,0,0,0]
        }]
        """
        let entries = parseHistory(from: json)
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries[0].date, "2026-04-17")
        XCTAssertEqual(entries[0].workMin, 280)
        XCTAssertEqual(entries[0].hourlyWork[10], 55)
    }

    func testParseLegacyEntryWithoutHourly() {
        let json = """
        [{
          "date": "2026-04-16",
          "work_min": 200,
          "break_min": 40,
          "sessions": 4,
          "activities": 2
        }]
        """
        let entries = parseHistory(from: json)
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries[0].hourlyWork, Array(repeating: 0, count: 24))
    }

    func testParseEmptyArray() {
        let entries = parseHistory(from: "[]")
        XCTAssertEqual(entries.count, 0)
    }

    func testParseMalformedReturnsEmpty() {
        let entries = parseHistory(from: "{not json")
        XCTAssertEqual(entries.count, 0)
    }
}
