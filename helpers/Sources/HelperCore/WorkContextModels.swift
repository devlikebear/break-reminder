import Foundation

/// One calendar event as sent to the engine (`workcontext.Event`).
public struct CalendarEventPayload: Codable, Equatable {
    public var id: String
    public var title: String
    public var start: Int64
    public var end: Int64
    public var allDay: Bool
    public var hasAttendees: Bool
    public var availability: String
    public var calendarID: String

    public init(id: String, title: String, start: Int64, end: Int64, allDay: Bool,
                hasAttendees: Bool, availability: String, calendarID: String) {
        self.id = id; self.title = title; self.start = start; self.end = end; self.allDay = allDay
        self.hasAttendees = hasAttendees; self.availability = availability; self.calendarID = calendarID
    }

    enum CodingKeys: String, CodingKey {
        case id, title, start, end, availability
        case allDay = "all_day", hasAttendees = "has_attendees", calendarID = "calendar_id"
    }

    /// JSON-object form for `EngineClient.call` params.
    public var dictionary: [String: Any] {
        ["id": id, "title": title, "start": start, "end": end, "all_day": allDay,
         "has_attendees": hasAttendees, "availability": availability, "calendar_id": calendarID]
    }
}

public struct AppUsageItem: Codable, Equatable {
    public var bundleID: String
    public var name: String
    public var seconds: Int
    public init(bundleID: String, name: String, seconds: Int) {
        self.bundleID = bundleID; self.name = name; self.seconds = seconds
    }
    enum CodingKeys: String, CodingKey { case name, seconds; case bundleID = "bundle_id" }
    public var dictionary: [String: Any] { ["bundle_id": bundleID, "name": name, "seconds": seconds] }
}

public struct CalendarContextStatus: Decodable, Equatable {
    public var current: CalendarEventPayload?
    public var next: CalendarEventPayload?
    public var focusMinutes = 0
    public var focusPomodoros = 0
    public var stale = true
    public var updatedAt: Int64 = 0
    public init() {}
    enum CodingKeys: String, CodingKey {
        case current, next, stale
        case focusMinutes = "focus_minutes", focusPomodoros = "focus_pomodoros", updatedAt = "updated_at"
    }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        current = try c.decodeIfPresent(CalendarEventPayload.self, forKey: .current)
        next = try c.decodeIfPresent(CalendarEventPayload.self, forKey: .next)
        focusMinutes = try c.decodeIfPresent(Int.self, forKey: .focusMinutes) ?? 0
        focusPomodoros = try c.decodeIfPresent(Int.self, forKey: .focusPomodoros) ?? 0
        stale = try c.decodeIfPresent(Bool.self, forKey: .stale) ?? true
        updatedAt = try c.decodeIfPresent(Int64.self, forKey: .updatedAt) ?? 0
    }
}

/// `status.context` from the engine.
public struct WorkContextStatus: Decodable, Equatable {
    public var calendar = CalendarContextStatus()
    public var appUsageToday: [AppUsageItem] = []
    public var appUsageEnabled = true
    public var calendarAwareBreaks = false
    public var meetingRule = "attendees"
    public var calendarExcludedIDs: [String] = []
    public init() {}
    enum CodingKeys: String, CodingKey {
        case calendar
        case appUsageToday = "app_usage_today", appUsageEnabled = "app_usage_enabled"
        case calendarAwareBreaks = "calendar_aware_breaks", meetingRule = "meeting_rule"
        case calendarExcludedIDs = "calendar_excluded_ids"
    }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        calendar = try c.decodeIfPresent(CalendarContextStatus.self, forKey: .calendar) ?? CalendarContextStatus()
        appUsageToday = try c.decodeIfPresent([AppUsageItem].self, forKey: .appUsageToday) ?? []
        appUsageEnabled = try c.decodeIfPresent(Bool.self, forKey: .appUsageEnabled) ?? true
        calendarAwareBreaks = try c.decodeIfPresent(Bool.self, forKey: .calendarAwareBreaks) ?? false
        meetingRule = try c.decodeIfPresent(String.self, forKey: .meetingRule) ?? "attendees"
        calendarExcludedIDs = try c.decodeIfPresent([String].self, forKey: .calendarExcludedIDs) ?? []
    }
}

/// Text for the popover "Now" card.
public struct NowCardLines: Equatable {
    public var headline: String
    public var focus: String?
    public var canStartFocus: Bool
}

public func nowCardLines(context: WorkContextStatus, calendarAuthorized: Bool, now: Int64) -> NowCardLines {
    guard calendarAuthorized else {
        return NowCardLines(headline: L10n.text("Connect your calendar to see your next event"), focus: nil, canStartFocus: false)
    }
    let cal = context.calendar
    if cal.stale {
        return NowCardLines(headline: L10n.text("Calendar data is out of date"), focus: nil, canStartFocus: false)
    }
    if let current = cal.current {
        let left = max(0, Int(current.end - now) / 60)
        return NowCardLines(headline: L10n.text("In a meeting: {0} · {1} min left", current.title, left), focus: nil, canStartFocus: false)
    }
    let headline: String
    if let next = cal.next {
        headline = L10n.text("Next: {0} · in {1}", next.title, untilText(seconds: next.start - now))
    } else {
        headline = L10n.text("No more events today")
    }
    // The window ends at the next event only when that event comes first;
    // otherwise it is the rest of today's work window.
    let boundedByEvent = cal.next.map { Int($0.start - now) / 60 <= cal.focusMinutes } ?? false
    guard boundedByEvent else {
        return NowCardLines(headline: headline, focus: L10n.text("{0} min of focus time left today", cal.focusMinutes), canStartFocus: false)
    }
    var focus = L10n.text("{0} min until the next event", cal.focusMinutes)
    if cal.focusPomodoros > 0 {
        focus += " · " + L10n.text("{0} pomodoros", cal.focusPomodoros)
    }
    return NowCardLines(headline: headline, focus: focus, canStartFocus: cal.focusPomodoros > 0)
}

private func untilText(seconds: Int64) -> String {
    let minutes = max(0, Int(seconds) / 60)
    if minutes >= 60 {
        return L10n.text("{0}h {1}m", minutes / 60, minutes % 60)
    }
    return L10n.text("{0} min", minutes)
}

/// "1h 2m" / "45m" / "<1m" for app usage rows.
public func formatUsageDuration(_ seconds: Int) -> String {
    seconds < 60 ? L10n.text("<1m") : formatMinutes(seconds / 60)
}

/// Accumulates foreground time per app between drains. Time while suspended
/// (screen locked, asleep, idle) is not counted.
public struct AppUsageAggregator {
    private var current: (bundleID: String, name: String)?
    private var segmentStart: Int64 = 0
    private var suspended = false
    private var pending: [String: AppUsageItem] = [:]
    private var order: [String] = []

    public init() {}

    public mutating func activate(bundleID: String, name: String, at: Int64) {
        close(at: at)
        current = (bundleID, name)
        segmentStart = at
    }

    public mutating func suspend(at: Int64) {
        guard !suspended else { return }
        close(at: at)
        suspended = true
    }

    public mutating func resume(at: Int64) {
        guard suspended else { return }
        suspended = false
        segmentStart = at
    }

    public var isSuspended: Bool { suspended }

    /// Returns increments since the previous drain, most used first.
    public mutating func drain(at: Int64) -> [AppUsageItem] {
        close(at: at)
        let items = order.compactMap { pending[$0] }.filter { $0.seconds > 0 }
            .sorted { $0.seconds != $1.seconds ? $0.seconds > $1.seconds : $0.bundleID < $1.bundleID }
        pending.removeAll(); order.removeAll()
        return items
    }

    private mutating func close(at: Int64) {
        guard let app = current, !suspended else { return }
        let seconds = Int(max(0, at - segmentStart))
        segmentStart = at
        guard seconds > 0 else { return }
        if pending[app.bundleID] == nil { order.append(app.bundleID) }
        var item = pending[app.bundleID] ?? AppUsageItem(bundleID: app.bundleID, name: app.name, seconds: 0)
        item.seconds += seconds
        item.name = app.name
        pending[app.bundleID] = item
    }
}

/// `YYYY-MM-DD.json`, matching the engine's app usage store.
public func appUsageFileName(for day: Date) -> String {
    let f = DateFormatter()
    f.calendar = Calendar(identifier: .gregorian)
    f.locale = Locale(identifier: "en_US_POSIX")
    f.dateFormat = "yyyy-MM-dd"
    return f.string(from: day) + ".json"
}

/// Reads one day's app usage written by the engine, most used first.
public func loadAppUsageTop(directory: URL, day: Date, limit: Int) -> [AppUsageItem] {
    guard let data = try? Data(contentsOf: directory.appendingPathComponent(appUsageFileName(for: day))),
          let items = try? JSONDecoder().decode([AppUsageItem].self, from: data) else { return [] }
    return Array(items.sorted { $0.seconds > $1.seconds }.prefix(limit))
}
