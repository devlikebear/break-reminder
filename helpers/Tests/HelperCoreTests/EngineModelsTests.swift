import XCTest
@testable import HelperCore

final class EngineModelsTests: XCTestCase {
    private let statusJSON = """
    {"now":1792458120,"state":{"work_seconds":120,"mode":"break","last_check":1792458100,"break_start":1792458000,
    "snooze_until":0,"paused":true,"paused_at":1792458110,"pause_reason":"focus","pause_until":0,
    "today_work_seconds":3000,"today_break_seconds":600,"last_update_date":"2026-10-20","last_break_warning_bucket":0,
    "hourly_work":[0,0,0,0,0,0,0,0,0,0,60,0,0,0,0,0,0,0,0,0,0,0,0,0],"session_state":"active","session_start":1792450000,
    "session_end":0,"session_manual":true,"last_activity":1792458100,"pomodoro_count":2,"today_pomodoros":3},
    "active":true,"timer_mode":"pomodoro","work_duration_sec":1500,"break_duration_sec":300,
    "pomodoro_work_min":25,"pomodoro_break_min":5,"pomodoro_long_break_min":15,"pomodoro_long_break_every":4,
    "idle_sec":4,"idle_threshold_sec":120,"break_screen_mode":"block","notifications_enabled":true,
    "time_tools":{"schema_version":1,"revision":7,"countdown":null,"recent_countdowns":[],"events":[]}}
    """

    func testStatusMapsOntoAppStateAndConfig() throws {
        let status = try JSONDecoder().decode(EngineStatus.self, from: Data(statusJSON.utf8))
        let s = status.appState
        XCTAssertEqual(s.mode, "break")
        XCTAssertEqual(s.workSeconds, 120)
        XCTAssertTrue(s.paused)
        XCTAssertEqual(s.pauseReason, "focus")
        XCTAssertTrue(s.isSessionActive)
        XCTAssertTrue(s.sessionManual)
        XCTAssertEqual(s.pomodoroCount, 2)
        XCTAssertEqual(s.todayPomodoros, 3)
        XCTAssertEqual(s.todayWorkSeconds, 3000)
        let c = status.appConfig
        XCTAssertTrue(c.pomodoroEnabled)
        XCTAssertEqual(c.pomodoroWorkMin, 25)
        XCTAssertEqual(c.effectiveWorkMin, 25)
        XCTAssertEqual(c.breakScreenMode, "block")
        XCTAssertEqual(status.timeTools?.revision, 7)
        XCTAssertTrue(status.active)
    }

    func testInboundLinesAreClassified() throws {
        switch try EngineInbound.parse(Data(#"{"jsonrpc":"2.0","id":4,"result":{"v":1}}"#.utf8)) {
        case let .response(id, result, error):
            XCTAssertEqual(id, 4); XCTAssertNil(error)
            XCTAssertEqual(String(decoding: result ?? Data(), as: UTF8.self), #"{"v":1}"#)
        default: XCTFail("expected response")
        }
        switch try EngineInbound.parse(Data(#"{"jsonrpc":"2.0","id":5,"error":{"code":-32001,"message":"no break"}}"#.utf8)) {
        case let .response(id, _, error):
            XCTAssertEqual(id, 5); XCTAssertEqual(error?.code, -32001); XCTAssertEqual(error?.message, "no break")
        default: XCTFail("expected error response")
        }
        switch try EngineInbound.parse(Data(#"{"jsonrpc":"2.0","method":"event","params":{"type":"break_start","title":"Break Time!","screen_mode":"block","break_sec":600,"break_start":100,"today_work_min":50,"today_break_min":10}}"#.utf8)) {
        case let .event(e):
            XCTAssertEqual(e.type, "break_start"); XCTAssertEqual(e.screenMode, "block")
            XCTAssertEqual(e.breakSec, 600); XCTAssertEqual(e.todayWorkMin, 50)
        default: XCTFail("expected event")
        }
        switch try EngineInbound.parse(Data(#"{"jsonrpc":"2.0","method":"status","params":\#(statusJSON)}"#.utf8)) {
        case let .status(s): XCTAssertEqual(s.state.mode, "break")
        default: XCTFail("expected status")
        }
        XCTAssertThrowsError(try EngineInbound.parse(Data("not json".utf8)))
    }

    func testRequestEncoding() throws {
        let line = try EngineRequest.encode(id: 9, method: "timer.pause", params: ["reason": "meeting"])
        XCTAssertEqual(line.last, UInt8(ascii: "\n"))
        let object = try JSONSerialization.jsonObject(with: line) as? [String: Any]
        XCTAssertEqual(object?["jsonrpc"] as? String, "2.0")
        XCTAssertEqual(object?["id"] as? Int, 9)
        XCTAssertEqual(object?["method"] as? String, "timer.pause")
        XCTAssertEqual((object?["params"] as? [String: Any])?["reason"] as? String, "meeting")
    }

    func testLineBufferSplitsChunks() {
        var buffer = LineBuffer()
        XCTAssertEqual(buffer.append(Data("{\"a\":1}\n{\"b\"".utf8)).map { String(decoding: $0, as: UTF8.self) }, ["{\"a\":1}"])
        XCTAssertEqual(buffer.append(Data(":2}\n\n".utf8)).map { String(decoding: $0, as: UTF8.self) }, ["{\"b\":2}"])
        XCTAssertTrue(buffer.append(Data()).isEmpty)
    }

    func testRestartBackoffGrowsAndCaps() {
        XCTAssertEqual(engineRestartDelay(attempt: 0), 1)
        XCTAssertEqual(engineRestartDelay(attempt: 1), 2)
        XCTAssertEqual(engineRestartDelay(attempt: 3), 8)
        XCTAssertEqual(engineRestartDelay(attempt: 10), 30)
    }

    func testBreakScreenArgumentsUseRemainingTime() {
        var e = EngineEvent(type: "break_start")
        e.breakSec = 600; e.breakStart = 1000; e.todayWorkMin = 50; e.todayBreakMin = 10
        XCTAssertEqual(breakScreenArguments(event: e, now: 1100),
                       ["--duration", "500", "--skip-after", "120", "--work-min", "50", "--break-min", "10"])
        XCTAssertNil(breakScreenArguments(event: e, now: 1600))
    }

    func testVoiceMatching() {
        let voices = [
            SpeechVoiceInfo(identifier: "com.apple.voice.compact.ko-KR.Yuna", name: "Yuna", language: "ko-KR"),
            SpeechVoiceInfo(identifier: "com.apple.voice.compact.en-US.Samantha", name: "Samantha", language: "en-US"),
        ]
        XCTAssertEqual(matchSpeechVoice(named: "Yuna", in: voices, fallbackLanguage: "en"), "com.apple.voice.compact.ko-KR.Yuna")
        XCTAssertEqual(matchSpeechVoice(named: "samantha", in: voices, fallbackLanguage: "ko"), "com.apple.voice.compact.en-US.Samantha")
        XCTAssertEqual(matchSpeechVoice(named: "Nope", in: voices, fallbackLanguage: "ko"), "com.apple.voice.compact.ko-KR.Yuna")
        XCTAssertNil(matchSpeechVoice(named: "Nope", in: voices, fallbackLanguage: "fr"))
    }
}

final class SpeechRouteTests: XCTestCase {
    func testConfiguredEngineKeepsItsVoiceOutsideTheSandbox() {
        XCTAssertEqual(speechRoute(ttsEngine: "gemini", sandboxed: false), .engineCLI)
        XCTAssertEqual(speechRoute(ttsEngine: "kittentts", sandboxed: false), .engineCLI)
        XCTAssertEqual(speechRoute(ttsEngine: "say", sandboxed: false), .system)
        XCTAssertEqual(speechRoute(ttsEngine: nil, sandboxed: false), .system)
        XCTAssertEqual(speechRoute(ttsEngine: "gemini", sandboxed: true), .system)
    }
}
