import XCTest
@testable import HelperCore

final class InsightsLoaderTests: XCTestCase {
    func testParseValidReport() {
        let json = """
        {
          "generated_at": "2026-04-17T17:30:00+09:00",
          "daily_report": "오늘 요약",
          "patterns": [
            {"type":"warning","title":"T1","description":"D1","suggestion":"S1"},
            {"type":"info","title":"T2","description":"D2","suggestion":"S2"}
          ]
        }
        """
        let report = parseInsights(from: json)
        XCTAssertNotNil(report)
        XCTAssertEqual(report?.dailyReport, "오늘 요약")
        XCTAssertEqual(report?.patterns.count, 2)
        XCTAssertEqual(report?.patterns[0].type, "warning")
    }

    func testParseEmptyPatterns() {
        let json = """
        {"generated_at":"x","daily_report":"r","patterns":[]}
        """
        let report = parseInsights(from: json)
        XCTAssertNotNil(report)
        XCTAssertEqual(report?.patterns.count, 0)
    }

    func testParseInvalidReturnsNil() {
        XCTAssertNil(parseInsights(from: "not json"))
    }
}
