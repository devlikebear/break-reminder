import Foundation

/// Creates a repeating timer that continues firing while AppKit tracks an open menu.
@discardableResult
public func scheduleMenuBarTimer(
    interval: TimeInterval,
    runLoop: RunLoop = .main,
    action: @escaping () -> Void
) -> Timer {
    let timer = Timer(timeInterval: interval, repeats: true) { _ in
        action()
    }
    // `.eventTracking` is explicit because not every host/test run loop includes it
    // in its common-mode set before NSApplication finishes initialization.
    runLoop.add(timer, forMode: .common)
    runLoop.add(timer, forMode: RunLoop.Mode("NSEventTrackingRunLoopMode"))
    return timer
}
