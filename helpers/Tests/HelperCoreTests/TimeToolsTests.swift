import XCTest
@testable import HelperCore

final class TimeToolsTests: XCTestCase {
    private let fixture = """
    {"schema_version":1,"revision":2,"countdown":{"id":"a","label":"코드 리뷰","duration_ms":10000,"phase":"running","deadline_unix_ms":110000,"remaining_ms":null,"created_at":100000,"completed_at":null},"recent_countdowns":[],"events":[],"future":{"preserved":true}}
    """
    func testGoSnapshotAndClockProjection() throws {
        let snapshot = try TimeToolsSnapshot.decode(Data(fixture.utf8))
        XCTAssertEqual(snapshot.revision, 2)
        XCTAssertEqual(snapshot.countdown?.remaining(nowMS: 104001), 5999)
        XCTAssertEqual(snapshot.countdown?.timeText(nowMS: 104001), "00:06")
        XCTAssertEqual(snapshot.countdown?.timeText(nowMS: 999999), "00:00")
        XCTAssertEqual(timeToolsMenuTitle(snapshot: snapshot, nowMS: 104001), "⏱ 00:06")
    }
    func testUnsupportedSchemaAndCorruptAreNotEmptyState() {
        XCTAssertThrowsError(try TimeToolsSnapshot.decode(Data("{\"schema_version\":99}".utf8)))
        XCTAssertThrowsError(try TimeToolsSnapshot.decode(Data("broken".utf8)))
    }
    func testUnreadCompletionOverridesCountdown() throws {
        let json = fixture.replacingOccurrences(of: "\"events\":[]", with: "\"events\":[{\"id\":\"old:completed\",\"source_id\":\"old\",\"kind\":\"countdown\",\"label\":\"차\",\"due_at\":90000,\"delivery_state\":\"failed\",\"attempted_at\":91000,\"acknowledged_at\":null}]")
        let snapshot = try TimeToolsSnapshot.decode(Data(json.utf8))
        XCTAssertEqual(timeToolsMenuTitle(snapshot: snapshot, nowMS: 104001), "완료 1")
        XCTAssertTrue(timeToolsOverlayText(snapshot: snapshot).contains("차"))
    }
    func testMissingFileAndInvalidFileDiffer() throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let files = TimeToolsFiles(directory: directory)
        XCTAssertNil(try files.loadSnapshot().countdown)
        try Data("broken".utf8).write(to: files.snapshotURL)
        XCTAssertThrowsError(try files.loadSnapshot())
    }
}
