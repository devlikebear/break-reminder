import XCTest
@testable import HelperCore

final class MenuBarAnimationTests: XCTestCase {
    func testWorkingUsesSixFrameRunningCycle() {
        let state = AppState()
        let sequence = (0..<6).map { menuBarAnimation(state: state, tick: $0) }

        XCTAssertEqual(sequence.map(\.kind), Array(repeating: .running, count: 6))
        XCTAssertEqual(sequence.map(\.frameIndex), [0, 1, 2, 3, 4, 5])
        XCTAssertEqual(menuBarAnimation(state: state, tick: 6).frameIndex, 0)
    }

    func testBreakUsesSlowTwoFrameRestCycle() {
        var state = AppState()
        state.mode = "break"

        XCTAssertEqual(menuBarAnimation(state: state, tick: 0), .init(kind: .resting, frameIndex: 0))
        XCTAssertEqual(menuBarAnimation(state: state, tick: 1), .init(kind: .resting, frameIndex: 0))
        XCTAssertEqual(menuBarAnimation(state: state, tick: 2), .init(kind: .resting, frameIndex: 1))
        XCTAssertEqual(menuBarAnimation(state: state, tick: 4), .init(kind: .resting, frameIndex: 0))
    }

    func testPausedUsesSleepingFrame() {
        var state = AppState()
        state.paused = true

        XCTAssertEqual(menuBarAnimation(state: state, tick: 99), .init(kind: .sleeping, frameIndex: 0))
    }

    func testPausedBreakUsesSleepingFrame() {
        var state = AppState()
        state.mode = "break"
        state.paused = true

        XCTAssertEqual(menuBarAnimation(state: state, tick: 5), .init(kind: .sleeping, frameIndex: 0))
    }

    func testNegativeTickWrapsToValidRunningFrame() {
        let frame = menuBarAnimation(state: AppState(), tick: -1)

        XCTAssertEqual(frame, .init(kind: .running, frameIndex: 5))
    }

    func testAnimatedTitleRemovesLegacyEmoji() {
        var state = AppState()
        state.workSeconds = 60
        state.lastCheck = 1_000
        var config = AppConfig()
        config.workDurationMin = 50

        let presentation = menuBarPresentation(state: state, config: config, now: 1_000)

        XCTAssertEqual(presentation.title, "2% · 49m left")
        XCTAssertFalse(presentation.title.contains("🐹"))
    }
}
