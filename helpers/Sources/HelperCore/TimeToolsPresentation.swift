import Foundation

public func timeToolsDuration(_ milliseconds: Int64) -> String {
    let seconds = max(0, milliseconds + 999) / 1000
    if seconds >= 3600 { return String(format: "%02lld:%02lld:%02lld", seconds / 3600, seconds / 60 % 60, seconds % 60) }
    return String(format: "%02lld:%02lld", seconds / 60, seconds % 60)
}
public func timeToolsMenuTitle(snapshot: TimeToolsSnapshot, nowMS: Int64) -> String? {
    if !snapshot.unread.isEmpty { return "완료 \(snapshot.unread.count)" }
    guard let countdown = snapshot.countdown, countdown.phase == "running" else { return nil }
    return "⏱ \(countdown.timeText(nowMS: nowMS))"
}
public func timeToolsOverlayText(snapshot: TimeToolsSnapshot) -> String {
    guard let event = snapshot.unread.last else { return "" }
    let suffix = snapshot.unread.count > 1 ? " 외 \(snapshot.unread.count - 1)개" : ""
    return "타이머 완료: \(event.label)\(suffix) · 자세한 내용은 휴식 후 확인"
}
