import AppKit
import HelperCore
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    private let model = AppModel()
    private var executor: ActionExecutor!
    private var statusItem: StatusItemController!
    private let idle = IdleReporter()

    func applicationDidFinishLaunching(_ notification: Notification) {
        executor = ActionExecutor(model: model)
        executor.requestNotificationPermission()
        model.onEvent = { [weak self] event in self?.executor.handle(event) }
        statusItem = StatusItemController(model: model)
        model.onEngineStarted = { [weak self] in self?.model.reportIdle(IdleReporter.idleSeconds()) }
        model.start()
        idle.start { [weak self] seconds in self?.model.reportIdle(seconds) }
        if let path = ProcessInfo.processInfo.environment["BREAK_REMINDER_POPOVER_SNAPSHOT"] {
            writePopoverSnapshot(to: URL(fileURLWithPath: path))
        }
    }

    /// Development hook: renders the popover to a PNG once a status arrived.
    private func writePopoverSnapshot(to url: URL) {
        Timer.scheduledTimer(withTimeInterval: 3, repeats: false) { [weak self] _ in
            Task { @MainActor in
                guard let self else { return }
                let view = NSHostingView(rootView: PopoverRootView(model: self.model))
                view.frame = NSRect(origin: .zero, size: view.fittingSize)
                let window = NSWindow(contentRect: view.frame, styleMask: [.borderless], backing: .buffered, defer: false)
                window.contentView = view
                view.layoutSubtreeIfNeeded()
                if let rep = view.bitmapImageRepForCachingDisplay(in: view.bounds) {
                    view.cacheDisplay(in: view.bounds, to: rep)
                    try? rep.representation(using: .png, properties: [:])?.write(to: url)
                }
            }
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        executor.stopOverlay()
        model.stop()
    }
}

MainActor.assumeIsolated {
    let app = NSApplication.shared
    app.setActivationPolicy(.accessory)
    let delegate = AppDelegate()
    app.delegate = delegate
    withExtendedLifetime(delegate) { app.run() }
}
