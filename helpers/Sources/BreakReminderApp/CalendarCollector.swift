import AppKit
import EventKit
import HelperCore

/// Reads today's and tomorrow's events with EventKit and sends them to the
/// engine on every calendar change and every 5 minutes. Asking for access is
/// always the user's click, never automatic.
@MainActor
final class CalendarCollector: ObservableObject {
    struct CalendarInfo: Identifiable, Equatable {
        let id: String
        let title: String
    }

    @Published private(set) var authorization = EKEventStore.authorizationStatus(for: .event)
    @Published private(set) var calendars: [CalendarInfo] = []
    @Published private(set) var lastSent: Date?
    @Published private(set) var lastError: String?

    private let store = EKEventStore()
    private let model: AppModel
    private var timer: Timer?
    private var observer: NSObjectProtocol?

    init(model: AppModel) { self.model = model }

    var authorized: Bool { authorization == .fullAccess }

    func start() {
        observer = NotificationCenter.default.addObserver(forName: .EKEventStoreChanged, object: store, queue: .main) { [weak self] _ in
            Task { @MainActor in self?.sync() }
        }
        timer = Timer.scheduledTimer(withTimeInterval: 300, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.sync() }
        }
        sync()
    }

    func requestAccess() {
        NSApp.activate(ignoringOtherApps: true)
        Task {
            _ = try? await store.requestFullAccessToEvents()
            authorization = EKEventStore.authorizationStatus(for: .event)
            sync()
        }
    }

    func openPrivacySettings() {
        if let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Calendars") {
            NSWorkspace.shared.open(url)
        }
    }

    func sync() {
        authorization = EKEventStore.authorizationStatus(for: .event)
        guard authorized else { return }
        calendars = store.calendars(for: .event)
            .map { CalendarInfo(id: $0.calendarIdentifier, title: $0.title) }
            .sorted { $0.title.localizedCompare($1.title) == .orderedAscending }

        let start = Calendar.current.startOfDay(for: Date())
        let end = Calendar.current.date(byAdding: .day, value: 2, to: start) ?? start.addingTimeInterval(172_800)
        let events = store.events(matching: store.predicateForEvents(withStart: start, end: end, calendars: nil))
        let payload = events.map { event -> [String: Any] in
            CalendarEventPayload(
                id: event.calendarItemIdentifier + "@" + String(Int64(event.startDate.timeIntervalSince1970)),
                title: event.title ?? "",
                start: Int64(event.startDate.timeIntervalSince1970),
                end: Int64(event.endDate.timeIntervalSince1970),
                allDay: event.isAllDay,
                hasAttendees: event.hasAttendees,
                availability: Self.availability(event.availability),
                calendarID: event.calendar.calendarIdentifier
            ).dictionary
        }
        Task {
            do {
                try await model.call("context.calendar.update", ["events": payload])
                lastSent = Date()
                lastError = nil
            } catch {
                lastError = error.localizedDescription
            }
        }
    }

    private static func availability(_ value: EKEventAvailability) -> String {
        switch value {
        case .busy: return "busy"
        case .free: return "free"
        case .tentative: return "tentative"
        case .unavailable: return "unavailable"
        default: return ""
        }
    }
}
