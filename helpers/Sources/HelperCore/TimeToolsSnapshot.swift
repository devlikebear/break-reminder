import Foundation
import Darwin

public struct CountdownSnapshot: Codable, Identifiable, Equatable {
    public var id: String
    public var label: String
    public var durationMS: Int64
    public var phase: String
    public var deadline: Int64?
    public var remainingMS: Int64?
    public var createdAt: Int64
    public var completedAt: Int64?
    enum CodingKeys: String, CodingKey {
        case id, label, phase
        case durationMS = "duration_ms", deadline = "deadline_unix_ms", remainingMS = "remaining_ms"
        case createdAt = "created_at", completedAt = "completed_at"
    }
    public func remaining(nowMS: Int64) -> Int64 {
        phase == "running" ? max(0, (deadline ?? nowMS) - nowMS) : max(0, remainingMS ?? 0)
    }
    public func timeText(nowMS: Int64) -> String { timeToolsDuration(remaining(nowMS: nowMS)) }
    public var isActive: Bool { phase == "running" || phase == "paused" }
}
public struct RecentCountdown: Codable, Equatable {
    public var label: String
    public var durationMS: Int64
    enum CodingKeys: String, CodingKey { case label; case durationMS = "duration_ms" }
}
public struct TimeToolsEvent: Codable, Identifiable, Equatable {
    public var id: String
    public var sourceID: String
    public var kind: String
    public var label: String
    public var dueAt: Int64
    public var deliveryState: String
    public var attemptedAt: Int64?
    public var acknowledgedAt: Int64?
    public var errorCode: String?
    enum CodingKeys: String, CodingKey {
        case id, kind, label
        case sourceID = "source_id", dueAt = "due_at", deliveryState = "delivery_state"
        case attemptedAt = "attempted_at", acknowledgedAt = "acknowledged_at", errorCode = "error_code"
    }
}
public struct TimeToolsSnapshot: Codable, Equatable {
    public var schemaVersion: Int = 1
    public var revision: UInt64 = 0
    public var countdown: CountdownSnapshot?
    public var recent: [RecentCountdown] = []
    public var events: [TimeToolsEvent] = []
    public init() {}
    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version", revision, countdown, recent = "recent_countdowns", events
    }
    public var unread: [TimeToolsEvent] { events.filter { $0.acknowledgedAt == nil } }
    public static func decode(_ data: Data) throws -> Self {
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        guard object?["schema_version"] as? Int == 1 else { throw TimeToolsReadError.unsupported }
        return try JSONDecoder().decode(Self.self, from: data)
    }
}
public enum TimeToolsReadError: LocalizedError {
    case unsupported
    public var errorDescription: String? { "시간 도구 데이터 버전을 지원하지 않습니다. 앱을 업데이트하세요." }
}
public struct TimeToolsRuntime: Codable {
    public var schemaVersion: Int = 1
    public var heartbeat: Int64 = 0
    public var pid: Int32 = 0
    public var ready: Bool = false
    public var notificationAvailable: Bool = false
    public var error: String?
    public init() {}
    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version", heartbeat = "heartbeat_unix_ms", pid, ready, error
        case notificationAvailable = "notification_available"
    }
    public func isRunning(nowMS: Int64) -> Bool {
        schemaVersion == 1 && ready && pid > 0 && nowMS >= heartbeat && nowMS - heartbeat <= 15_000 && kill(pid, 0) == 0
    }
}
public struct TimeToolsFiles {
    public let directory: URL
    public init(directory: URL? = nil) {
        if let directory { self.directory = directory }
        else if let testPath = ProcessInfo.processInfo.environment["BREAK_REMINDER_TIME_TOOLS_DIR"] {
            self.directory = URL(fileURLWithPath: testPath, isDirectory: true)
        } else {
            self.directory = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".config/break-reminder", isDirectory: true)
        }
    }
    public var snapshotURL: URL { directory.appendingPathComponent("time-tools.json") }
    public func loadSnapshot() throws -> TimeToolsSnapshot {
        do { return try TimeToolsSnapshot.decode(Data(contentsOf: snapshotURL)) }
        catch let error as NSError where error.domain == NSCocoaErrorDomain && error.code == NSFileReadNoSuchFileError { return TimeToolsSnapshot() }
    }
    public func loadRuntime() -> TimeToolsRuntime {
        guard let data = try? Data(contentsOf: directory.appendingPathComponent("time-tools-runtime.json")),
              let value = try? JSONDecoder().decode(TimeToolsRuntime.self, from: data) else { return TimeToolsRuntime() }
        return value
    }
    public func arguments(_ command: [String]) -> [String] { ["time-tools", "--data-dir", directory.path, "--json"] + command }
}
