import XCTest
@testable import HelperCore

final class WorkContextTests: XCTestCase {
    override func setUp() { setenv("BREAK_REMINDER_LANGUAGE", "en", 1) }

    func testCalendarPayloadEncodesEngineKeys() throws {
        let e = CalendarEventPayload(id: "x", title: "Standup", start: 100, end: 200, allDay: false,
                                     hasAttendees: true, availability: "busy", calendarID: "work")
        let object = try JSONSerialization.jsonObject(with: JSONEncoder().encode(e)) as? [String: Any]
        XCTAssertEqual(object?["has_attendees"] as? Bool, true)
        XCTAssertEqual(object?["calendar_id"] as? String, "work")
        XCTAssertEqual(object?["all_day"] as? Bool, false)
        XCTAssertEqual(object?["start"] as? Int, 100)
        XCTAssertEqual(e.dictionary["availability"] as? String, "busy")
    }

    func testStatusDecodesContext() throws {
        let json = """
        {"calendar":{"current":{"id":"m","title":"Standup","start":0,"end":600,"all_day":false,"has_attendees":true,"availability":"busy","calendar_id":"c"},
        "focus_minutes":0,"focus_pomodoros":0,"stale":false,"updated_at":5},
        "app_usage_today":[{"bundle_id":"com.apple.Safari","name":"Safari","seconds":3720}],
        "app_usage_enabled":true,"calendar_aware_breaks":false,"meeting_rule":"attendees","calendar_excluded_ids":null}
        """
        let ctx = try JSONDecoder().decode(WorkContextStatus.self, from: Data(json.utf8))
        XCTAssertEqual(ctx.calendar.current?.title, "Standup")
        XCTAssertEqual(ctx.appUsageToday.first?.name, "Safari")
        XCTAssertEqual(ctx.calendarExcludedIDs, [])
    }

    func testNowCardLines() {
        let now: Int64 = 1_000_000
        var ctx = WorkContextStatus()
        ctx.calendar.updatedAt = now; ctx.calendar.stale = false
        XCTAssertEqual(nowCardLines(context: ctx, calendarAuthorized: false, now: now).headline, L10n.text("Connect your calendar to see your next event"))

        ctx.calendar.next = CalendarEventPayload(id: "n", title: "Review", start: now + 600, end: now + 1800, allDay: false, hasAttendees: true, availability: "busy", calendarID: "c")
        ctx.calendar.focusMinutes = 10
        let soon = nowCardLines(context: ctx, calendarAuthorized: true, now: now)
        XCTAssertEqual(soon.headline, "Next: Review · in 10 min")
        XCTAssertEqual(soon.focus, "10 min until the next event")
        XCTAssertFalse(soon.canStartFocus)

        ctx.calendar.focusMinutes = 55; ctx.calendar.focusPomodoros = 2
        let focus = nowCardLines(context: ctx, calendarAuthorized: true, now: now)
        XCTAssertEqual(focus.focus, "55 min until the next event · 2 pomodoros")
        XCTAssertTrue(focus.canStartFocus)

        ctx.calendar.current = CalendarEventPayload(id: "m", title: "Standup", start: now - 60, end: now + 840, allDay: false, hasAttendees: true, availability: "busy", calendarID: "c")
        XCTAssertEqual(nowCardLines(context: ctx, calendarAuthorized: true, now: now).headline, "In a meeting: Standup · 14 min left")

        ctx.calendar.stale = true
        XCTAssertEqual(nowCardLines(context: ctx, calendarAuthorized: true, now: now).headline, L10n.text("Calendar data is out of date"))
    }

    func testFocusBoundedByEndOfDayIsNotCalledTimeUntilTheEvent() {
        let now: Int64 = 1_000_000
        var ctx = WorkContextStatus()
        ctx.calendar.updatedAt = now; ctx.calendar.stale = false
        // Next event is tomorrow; the focus window ends with today's detection window.
        ctx.calendar.next = CalendarEventPayload(id: "n", title: "Tomorrow", start: now + 21 * 3600, end: now + 22 * 3600, allDay: false, hasAttendees: true, availability: "busy", calendarID: "c")
        ctx.calendar.focusMinutes = 674; ctx.calendar.focusPomodoros = 22
        let lines = nowCardLines(context: ctx, calendarAuthorized: true, now: now)
        XCTAssertEqual(lines.focus, "674 min of focus time left today")
        XCTAssertFalse(lines.canStartFocus)
    }

    func testAppUsageAggregatorSplitsByAppAndSkipsSuspendedTime() {
        var agg = AppUsageAggregator()
        agg.activate(bundleID: "com.apple.Safari", name: "Safari", at: 0)
        agg.activate(bundleID: "com.microsoft.VSCode", name: "Code", at: 30)
        agg.suspend(at: 50)          // screen locked
        agg.resume(at: 500)
        let first = agg.drain(at: 560)
        XCTAssertEqual(first.map(\.bundleID), ["com.microsoft.VSCode", "com.apple.Safari"])
        XCTAssertEqual(first.first?.seconds, 80) // 20 before lock + 60 after
        XCTAssertEqual(first.last?.seconds, 30)
        // Drain only reports what happened since the previous drain.
        let second = agg.drain(at: 620)
        XCTAssertEqual(second.count, 1)
        XCTAssertEqual(second.first?.seconds, 60)
        XCTAssertTrue(agg.drain(at: 620).isEmpty)
    }

    func testFormatUsageDuration() {
        XCTAssertEqual(formatUsageDuration(3720), "1h 2m")
        XCTAssertEqual(formatUsageDuration(59), "<1m")
    }
}

final class AppUsageFileTests: XCTestCase {
    func testLoadsDayFileAndTopFive() throws {
        let dir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: dir) }
        let items = (1...7).map { AppUsageItem(bundleID: "app\($0)", name: "App \($0)", seconds: $0 * 60) }
        let day = Date(timeIntervalSince1970: 1_792_458_000)
        try JSONEncoder().encode(items).write(to: dir.appendingPathComponent(appUsageFileName(for: day)))
        let top = loadAppUsageTop(directory: dir, day: day, limit: 5)
        XCTAssertEqual(top.count, 5)
        XCTAssertEqual(top.first?.bundleID, "app7")
        XCTAssertTrue(loadAppUsageTop(directory: dir, day: day.addingTimeInterval(86_400), limit: 5).isEmpty)
    }
}
