import Foundation

/// Timer state as serialized by the Go engine (`state.State` JSON tags).
public struct EngineState: Codable, Equatable {
    public var workSeconds: Int = 0
    public var mode: String = "work"
    public var lastCheck: Int64 = 0
    public var breakStart: Int64 = 0
    public var snoozeUntil: Int64 = 0
    public var paused: Bool = false
    public var pausedAt: Int64 = 0
    public var pauseReason: String = ""
    public var pauseUntil: Int64 = 0
    public var todayWorkSeconds: Int = 0
    public var todayBreakSeconds: Int = 0
    public var lastUpdateDate: String = ""
    public var sessionState: String = ""
    public var sessionStart: Int64 = 0
    public var sessionEnd: Int64 = 0
    public var sessionManual: Bool = false
    public var lastActivity: Int64 = 0
    public var pomodoroCount: Int = 0
    public var todayPomodoros: Int = 0

    public init() {}

    enum CodingKeys: String, CodingKey {
        case mode, paused
        case workSeconds = "work_seconds", lastCheck = "last_check", breakStart = "break_start"
        case snoozeUntil = "snooze_until", pausedAt = "paused_at", pauseReason = "pause_reason"
        case pauseUntil = "pause_until", todayWorkSeconds = "today_work_seconds"
        case todayBreakSeconds = "today_break_seconds", lastUpdateDate = "last_update_date"
        case sessionState = "session_state", sessionStart = "session_start", sessionEnd = "session_end"
        case sessionManual = "session_manual", lastActivity = "last_activity"
        case pomodoroCount = "pomodoro_count", todayPomodoros = "today_pomodoros"
    }
}

/// Snapshot published by the engine after every tick and change.
public struct EngineStatus: Decodable, Equatable {
    public var now: Int64
    public var state: EngineState
    public var active: Bool
    public var timerMode: String
    public var workDurationSec: Int
    public var breakDurationSec: Int
    public var pomodoroWorkMin: Int
    public var pomodoroBreakMin: Int
    public var pomodoroLongBreakMin: Int
    public var pomodoroLongBreakEvery: Int
    public var idleSec: Int
    public var idleThresholdSec: Int
    public var breakScreenMode: String
    public var notificationsEnabled: Bool
    public var autoSessionDetect: Bool?
    public var checkIntervalSec: Int?
    public var timeTools: TimeToolsSnapshot?
    public var configError: String?
    public var context: WorkContextStatus?
    public var ttsEngine: String?
    public var assistant: AssistantDay?

    enum CodingKeys: String, CodingKey {
        case now, state, active
        case timerMode = "timer_mode", workDurationSec = "work_duration_sec", breakDurationSec = "break_duration_sec"
        case pomodoroWorkMin = "pomodoro_work_min", pomodoroBreakMin = "pomodoro_break_min"
        case pomodoroLongBreakMin = "pomodoro_long_break_min", pomodoroLongBreakEvery = "pomodoro_long_break_every"
        case idleSec = "idle_sec", idleThresholdSec = "idle_threshold_sec", breakScreenMode = "break_screen_mode"
        case notificationsEnabled = "notifications_enabled", autoSessionDetect = "auto_session_detect"
        case checkIntervalSec = "check_interval_sec", timeTools = "time_tools", configError = "config_error"
        case context, assistant, ttsEngine = "tts_engine"
    }

    /// The state in the shape the shared presentation helpers expect.
    public var appState: AppState {
        var s = AppState()
        s.workSeconds = state.workSeconds; s.mode = state.mode; s.lastCheck = state.lastCheck
        s.breakStart = state.breakStart; s.snoozeUntil = state.snoozeUntil; s.paused = state.paused
        s.pausedAt = state.pausedAt; s.pauseReason = state.pauseReason; s.pauseUntil = state.pauseUntil
        s.todayWorkSeconds = state.todayWorkSeconds; s.todayBreakSeconds = state.todayBreakSeconds
        s.lastUpdateDate = state.lastUpdateDate; s.sessionState = state.sessionState
        s.sessionStart = state.sessionStart; s.sessionEnd = state.sessionEnd; s.sessionManual = state.sessionManual
        s.lastActivity = state.lastActivity; s.pomodoroCount = state.pomodoroCount; s.todayPomodoros = state.todayPomodoros
        return s
    }

    /// The config subset the presentation helpers read. Durations come from
    /// the engine's effective values so classic and pomodoro both render right.
    public var appConfig: AppConfig {
        var c = AppConfig()
        c.timerMode = timerMode
        c.pomodoroWorkMin = pomodoroWorkMin; c.pomodoroBreakMin = pomodoroBreakMin
        c.pomodoroLongBreakMin = pomodoroLongBreakMin; c.pomodoroLongBreakEvery = pomodoroLongBreakEvery
        if timerMode != "pomodoro" {
            c.workDurationMin = workDurationSec / 60
            c.breakDurationMin = breakDurationSec / 60
        }
        c.idleThresholdSec = idleThresholdSec
        c.breakScreenMode = breakScreenMode
        c.notificationsEnabled = notificationsEnabled
        c.autoSessionDetect = autoSessionDetect ?? true
        c.checkIntervalSec = checkIntervalSec ?? 60
        return c
    }
}

/// A reminder the app should show or say (`internal/reminder.Event`).
public struct EngineEvent: Codable, Equatable {
    public var type: String
    public var title: String?
    public var body: String?
    public var sound: String?
    public var text: String?
    public var voice: String?
    public var screenMode: String?
    public var breakSec: Int?
    public var breakStart: Int64?
    public var todayWorkMin: Int?
    public var todayBreakMin: Int?
    public var id: String?

    public init(type: String) { self.type = type }

    enum CodingKeys: String, CodingKey {
        case type, title, body, sound, text, voice, id
        case screenMode = "screen_mode", breakSec = "break_sec", breakStart = "break_start"
        case todayWorkMin = "today_work_min", todayBreakMin = "today_break_min"
    }
}

public struct EngineRPCError: Error, Equatable, LocalizedError {
    public var code: Int
    public var message: String
    public init(code: Int, message: String) { self.code = code; self.message = message }
    public var errorDescription: String? { message }
}

/// One line read from the engine's stdout.
public enum EngineInbound {
    case response(id: Int, result: Data?, error: EngineRPCError?)
    case event(EngineEvent)
    case status(EngineStatus)
    case agentDelta(requestID: String, text: String)
    case other

    public static func parse(_ line: Data) throws -> EngineInbound {
        guard let object = try JSONSerialization.jsonObject(with: line) as? [String: Any] else { return .other }
        if let method = object["method"] as? String {
            let params = try JSONSerialization.data(withJSONObject: object["params"] ?? [:])
            switch method {
            case "event": return .event(try JSONDecoder().decode(EngineEvent.self, from: params))
            case "status": return .status(try JSONDecoder().decode(EngineStatus.self, from: params))
            case "agent.delta":
                let p = object["params"] as? [String: Any] ?? [:]
                return .agentDelta(requestID: p["request_id"] as? String ?? "", text: p["text"] as? String ?? "")
            default: return .other
            }
        }
        guard let id = object["id"] as? Int else { return .other }
        if let error = object["error"] as? [String: Any] {
            return .response(id: id, result: nil, error: EngineRPCError(code: error["code"] as? Int ?? -32603, message: error["message"] as? String ?? ""))
        }
        let result = try object["result"].map { try JSONSerialization.data(withJSONObject: $0, options: [.fragmentsAllowed]) }
        return .response(id: id, result: result, error: nil)
    }
}

public enum EngineRequest {
    public static func encode(id: Int, method: String, params: [String: Any]) throws -> Data {
        var data = try JSONSerialization.data(withJSONObject: ["jsonrpc": "2.0", "id": id, "method": method, "params": params], options: [.sortedKeys])
        data.append(UInt8(ascii: "\n"))
        return data
    }
}

/// Splits a byte stream into newline-terminated lines.
public struct LineBuffer {
    private var pending = Data()
    public init() {}
    public mutating func append(_ chunk: Data) -> [Data] {
        pending.append(chunk)
        var lines: [Data] = []
        while let newline = pending.firstIndex(of: UInt8(ascii: "\n")) {
            let line = pending[pending.startIndex..<newline]
            if !line.isEmpty { lines.append(Data(line)) }
            pending = Data(pending[pending.index(after: newline)...])
        }
        return lines
    }
}

/// Seconds to wait before restarting a crashed engine.
public func engineRestartDelay(attempt: Int) -> TimeInterval {
    min(30, pow(2, Double(max(0, min(attempt, 10)))))
}

/// Arguments for the bundled `break-screen` helper, or nil when the break has
/// already run out.
public func breakScreenArguments(event: EngineEvent, now: Int64) -> [String]? {
    let breakSec = event.breakSec ?? 0
    let start = event.breakStart ?? now
    let remaining = breakSec - Int(now - start)
    guard remaining > 0 else { return nil }
    return ["--duration", String(remaining), "--skip-after", "120",
            "--work-min", String(event.todayWorkMin ?? 0), "--break-min", String(event.todayBreakMin ?? 0)]
}

public struct SpeechVoiceInfo: Equatable {
    public var identifier: String
    public var name: String
    public var language: String
    public init(identifier: String, name: String, language: String) {
        self.identifier = identifier; self.name = name; self.language = language
    }
}

/// Maps the configured `say` voice name onto an installed speech voice,
/// falling back to any voice of the UI language.
public func matchSpeechVoice(named name: String, in voices: [SpeechVoiceInfo], fallbackLanguage: String) -> String? {
    if let exact = voices.first(where: { $0.name.caseInsensitiveCompare(name) == .orderedSame }) {
        return exact.identifier
    }
    return voices.first(where: { $0.language.lowercased().hasPrefix(fallbackLanguage.lowercased()) })?.identifier
}

public enum SpeechRoute: Equatable {
    /// AVSpeechSynthesizer with the closest system voice.
    case system
    /// `engine tts test <text>`: the configured engine (Gemini, KittenTTS, …).
    case engineCLI
}

/// Picks how to speak. The direct build keeps the user's configured TTS
/// engine; the sandboxed build cannot run it and uses system voices.
public func speechRoute(ttsEngine: String?, sandboxed: Bool) -> SpeechRoute {
    guard !sandboxed, let engine = ttsEngine, !engine.isEmpty, engine != "say" else { return .system }
    return .engineCLI
}
