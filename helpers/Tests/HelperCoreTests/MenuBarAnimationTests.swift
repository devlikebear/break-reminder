import XCTest
@testable import HelperCore

final class MenuBarAnimationTests: XCTestCase {
    private func activeState() -> AppState {
        var state = AppState()
        state.sessionState = "active"
        return state
    }

    func testWorkingUsesSixFrameRunningCycle() {
        let state = activeState()
        let config = AppConfig()
        let sequence = (0..<6).map { menuBarAnimation(state: state, config: config, tick: $0) }

        XCTAssertEqual(sequence.map(\.kind), Array(repeating: .running, count: 6))
        XCTAssertEqual(sequence.map(\.frameIndex), [0, 1, 2, 3, 4, 5])
        XCTAssertEqual(menuBarAnimation(state: state, config: config, tick: 6).frameIndex, 0)
    }

    func testBreakUsesSlowTwoFrameRestCycle() {
        var state = activeState()
        state.mode = "break"
        let config = AppConfig()

        XCTAssertEqual(menuBarAnimation(state: state, config: config, tick: 0), .init(kind: .resting, frameIndex: 0))
        XCTAssertEqual(menuBarAnimation(state: state, config: config, tick: 1), .init(kind: .resting, frameIndex: 0))
        XCTAssertEqual(menuBarAnimation(state: state, config: config, tick: 2), .init(kind: .resting, frameIndex: 1))
        XCTAssertEqual(menuBarAnimation(state: state, config: config, tick: 4), .init(kind: .resting, frameIndex: 0))
    }

    func testPausedUsesSleepingFrame() {
        var state = activeState()
        state.paused = true

        XCTAssertEqual(menuBarAnimation(state: state, config: AppConfig(), tick: 99), .init(kind: .sleeping, frameIndex: 0))
    }

    func testStoppedSessionUsesSleepingFrame() {
        var state = AppState()
        state.sessionState = "ended"
        state.sessionManual = true

        XCTAssertEqual(menuBarAnimation(state: state, config: AppConfig(), tick: 3), .init(kind: .sleeping, frameIndex: 0))
    }

    func testLegacySchedulingKeepsRunningWithoutSession() {
        var config = AppConfig()
        config.autoSessionDetect = false

        XCTAssertEqual(menuBarAnimation(state: AppState(), config: config, tick: 0), .init(kind: .running, frameIndex: 0))
    }

    func testPausedBreakUsesSleepingFrame() {
        var state = activeState()
        state.mode = "break"
        state.paused = true

        XCTAssertEqual(menuBarAnimation(state: state, config: AppConfig(), tick: 5), .init(kind: .sleeping, frameIndex: 0))
    }

    func testNegativeTickWrapsToValidRunningFrame() {
        let frame = menuBarAnimation(state: activeState(), config: AppConfig(), tick: -1)

        XCTAssertEqual(frame, .init(kind: .running, frameIndex: 5))
    }

    func testAnimatedTitleRemovesLegacyEmoji() {
        var state = activeState()
        state.workSeconds = 60
        state.lastCheck = 1_000
        var config = AppConfig()
        config.workDurationMin = 50

        let presentation = menuBarPresentation(state: state, config: config, now: 1_000)

        XCTAssertEqual(presentation.title, "2% · 49m left")
        XCTAssertFalse(presentation.title.contains("🐹"))
    }
}
