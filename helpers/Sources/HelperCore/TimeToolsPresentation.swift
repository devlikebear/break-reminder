import Foundation

public func timeToolsDuration(_ milliseconds: Int64) -> String {
    let seconds = max(0, milliseconds + 999) / 1000
    if seconds >= 3600 { return String(format: "%02lld:%02lld:%02lld", seconds / 3600, seconds / 60 % 60, seconds % 60) }
    return String(format: "%02lld:%02lld", seconds / 60, seconds % 60)
}
public func timeToolsMenuTitle(snapshot: TimeToolsSnapshot, nowMS: Int64) -> String? {
    if !snapshot.unread.isEmpty { return L10n.text("Done {0}", snapshot.unread.count) }
    guard let countdown = snapshot.countdown, countdown.phase == "running" else { return nil }
    return "⏱ \(countdown.timeText(nowMS: nowMS))"
}
public func timeToolsOverlayText(snapshot: TimeToolsSnapshot) -> String {
    guard let event = snapshot.unread.last else { return "" }
    let suffix = snapshot.unread.count > 1 ? L10n.text(" (+{0} more)", snapshot.unread.count - 1) : ""
    return L10n.text("Timer complete: {0}{1} · check details after your break", event.displayLabel, suffix)
}
