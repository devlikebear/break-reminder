import XCTest
@testable import HelperCore

final class PopoverPresentationTests: XCTestCase {
    override func setUp() { setenv("BREAK_REMINDER_LANGUAGE", "en", 1) }

    private func status(_ edit: (inout EngineStatus) -> Void) -> EngineStatus {
        var s = EngineStatus(now: 0, state: EngineState(), active: true, timerMode: "classic",
                             workDurationSec: 3000, breakDurationSec: 600,
                             pomodoroWorkMin: 25, pomodoroBreakMin: 5, pomodoroLongBreakMin: 15, pomodoroLongBreakEvery: 4,
                             idleSec: 0, idleThresholdSec: 120, breakScreenMode: "block", notificationsEnabled: true)
        edit(&s)
        return s
    }

    func testWorkingSessionShowsProgress() {
        let now: Int64 = 1_800_000_000
        let st = status {
            $0.state.sessionState = "active"; $0.state.workSeconds = 1500; $0.state.lastCheck = now
        }
        let p = popoverPresentation(status: st, now: now)
        XCTAssertEqual(p.phase, .working)
        XCTAssertEqual(p.progress, 0.5, accuracy: 0.001)
        XCTAssertTrue(p.sessionActive)
        XCTAssertFalse(p.canSnooze)
        XCTAssertTrue(p.canSwitchMode)
        XCTAssertTrue(p.statusLine.contains("25m until break"), p.statusLine)
    }

    func testBreakAllowsSnooze() {
        let now: Int64 = 1_800_000_000
        let st = status {
            $0.state.sessionState = "active"; $0.state.mode = "break"; $0.state.breakStart = now - 300; $0.state.lastCheck = now
        }
        let p = popoverPresentation(status: st, now: now)
        XCTAssertEqual(p.phase, .onBreak)
        XCTAssertEqual(p.progress, 0.5, accuracy: 0.001)
        XCTAssertTrue(p.canSnooze)
        XCTAssertFalse(p.canSwitchMode)
    }

    func testPausedAndIdle() {
        let paused = popoverPresentation(status: status { $0.state.sessionState = "active"; $0.state.paused = true }, now: 100)
        XCTAssertEqual(paused.phase, .paused)
        XCTAssertFalse(paused.canSnooze)
        let idle = popoverPresentation(status: status { $0.state.sessionState = "" }, now: 100)
        XCTAssertEqual(idle.phase, .idle)
        XCTAssertFalse(idle.sessionActive)
        XCTAssertEqual(idle.progress, 0)
    }
}
