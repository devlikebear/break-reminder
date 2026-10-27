import AppKit
import HelperCore

/// Records which app is in front, by bundle ID and name only (never window
/// titles), and sends one-minute increments to the engine. Locked, asleep or
/// idle time does not count.
@MainActor
final class AppUsageCollector {
    private let model: AppModel
    private var aggregator = AppUsageAggregator()
    private var timer: Timer?
    private var observers: [(NotificationCenter, NSObjectProtocol)] = []

    init(model: AppModel) { self.model = model }

    private static var now: Int64 { Int64(Date().timeIntervalSince1970) }

    func start() {
        if let app = NSWorkspace.shared.frontmostApplication { activate(app) }
        let ws = NSWorkspace.shared.notificationCenter
        observe(ws, NSWorkspace.didActivateApplicationNotification) { [weak self] note in
            if let app = note.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication { self?.activate(app) }
        }
        for name in [NSWorkspace.willSleepNotification, NSWorkspace.screensDidSleepNotification, NSWorkspace.sessionDidResignActiveNotification] {
            observe(ws, name) { [weak self] _ in self?.aggregator.suspend(at: Self.now) }
        }
        for name in [NSWorkspace.didWakeNotification, NSWorkspace.screensDidWakeNotification, NSWorkspace.sessionDidBecomeActiveNotification] {
            observe(ws, name) { [weak self] _ in self?.aggregator.resume(at: Self.now) }
        }
        let distributed = DistributedNotificationCenter.default()
        observe(distributed, Notification.Name("com.apple.screenIsLocked")) { [weak self] _ in self?.aggregator.suspend(at: Self.now) }
        observe(distributed, Notification.Name("com.apple.screenIsUnlocked")) { [weak self] _ in self?.aggregator.resume(at: Self.now) }
        timer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.flush() }
        }
    }

    /// Called with each idle measurement: idle stretches are not app use.
    func noteIdle(_ seconds: Int) {
        let threshold = model.status?.idleThresholdSec ?? 120
        if seconds >= threshold {
            aggregator.suspend(at: Self.now - Int64(seconds))
        } else {
            aggregator.resume(at: Self.now - Int64(seconds))
        }
    }

    func flush() {
        let items = aggregator.drain(at: Self.now)
        guard !items.isEmpty, model.status?.context?.appUsageEnabled ?? true else { return }
        Task { try? await model.call("context.app_usage.add", ["items": items.map(\.dictionary)]) }
    }

    private func activate(_ app: NSRunningApplication) {
        guard let id = app.bundleIdentifier else { return }
        aggregator.activate(bundleID: id, name: app.localizedName ?? id, at: Self.now)
    }

    private func observe(_ center: NotificationCenter, _ name: Notification.Name, _ handler: @escaping (Notification) -> Void) {
        let token = center.addObserver(forName: name, object: nil, queue: .main) { note in
            MainActor.assumeIsolated { handler(note) }
        }
        observers.append((center, token))
    }
}
