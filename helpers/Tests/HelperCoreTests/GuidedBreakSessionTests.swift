import XCTest
@testable import HelperCore

final class GuidedBreakSessionTests: XCTestCase {
    func testInitialPhaseIsReady() {
        let session = GuidedBreakSession()

        XCTAssertEqual(session.phase, .ready)
        XCTAssertEqual(GuidedBreakSession.activityID, "standing-neck-shoulder-stretch-v1")
        XCTAssertEqual(GuidedBreakSession.activityDurationSeconds, 120)
        XCTAssertEqual(GuidedBreakSession.completionDisplaySeconds, 3)
    }

    func testTickDoesNotAutoStartReadySession() {
        var session = GuidedBreakSession()

        XCTAssertEqual(session.tick(), .stay)
        XCTAssertEqual(session.phase, .ready)
    }

    func testStartRequiresActivityAndCompletionBudget() {
        var shortSession = GuidedBreakSession()
        XCTAssertFalse(shortSession.start(availableBreakSeconds: 122))
        XCTAssertEqual(shortSession.phase, .ready)

        var exactSession = GuidedBreakSession()
        XCTAssertTrue(exactSession.start(availableBreakSeconds: 123))
        XCTAssertEqual(exactSession.phase, .running(remainingSeconds: 120))
    }

    func testStartOnlySucceedsFromReady() {
        var session = GuidedBreakSession()
        XCTAssertTrue(session.start(availableBreakSeconds: 123))

        XCTAssertFalse(session.start(availableBreakSeconds: 600))
        XCTAssertEqual(session.phase, .running(remainingSeconds: 120))
    }

    func testSixtyTicksLeavesSixtySeconds() {
        var session = GuidedBreakSession()
        XCTAssertTrue(session.start(availableBreakSeconds: 600))

        for _ in 0..<60 {
            XCTAssertEqual(session.tick(), .stay)
        }

        XCTAssertEqual(session.phase, .running(remainingSeconds: 60))
    }

    func testStartRunsForExactly120TicksThenShowsCompletionForThreeTicks() {
        var session = GuidedBreakSession()
        XCTAssertTrue(session.start(availableBreakSeconds: 600))

        for _ in 0..<119 {
            XCTAssertEqual(session.tick(), .stay)
        }
        XCTAssertEqual(session.phase, .running(remainingSeconds: 1))

        XCTAssertEqual(session.tick(), .phaseChanged)
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 3))

        XCTAssertEqual(session.tick(), .stay)
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 2))
        XCTAssertEqual(session.tick(), .stay)
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 1))
        XCTAssertEqual(session.tick(), .dismiss)
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 1))
    }

    func testInstructionTextAtAllBoundaries() {
        var session = GuidedBreakSession()
        XCTAssertTrue(session.start(availableBreakSeconds: 600))

        XCTAssertEqual(session.phase, .running(remainingSeconds: 120))
        XCTAssertEqual(session.instructionText(), "편안히 서서 어깨의 힘을 빼세요.")

        advance(&session, ticks: 39)
        XCTAssertEqual(session.phase, .running(remainingSeconds: 81))
        XCTAssertEqual(session.instructionText(), "편안히 서서 어깨의 힘을 빼세요.")

        advance(&session, ticks: 1)
        XCTAssertEqual(session.phase, .running(remainingSeconds: 80))
        XCTAssertEqual(session.instructionText(), "고개를 천천히 좌우로 기울이고, 통증이 있으면 멈추세요.")

        advance(&session, ticks: 39)
        XCTAssertEqual(session.phase, .running(remainingSeconds: 41))
        XCTAssertEqual(session.instructionText(), "고개를 천천히 좌우로 기울이고, 통증이 있으면 멈추세요.")

        advance(&session, ticks: 1)
        XCTAssertEqual(session.phase, .running(remainingSeconds: 40))
        XCTAssertEqual(session.instructionText(), "어깨를 뒤로 천천히 돌리며 호흡하세요.")

        advance(&session, ticks: 39)
        XCTAssertEqual(session.phase, .running(remainingSeconds: 1))
        XCTAssertEqual(session.instructionText(), "어깨를 뒤로 천천히 돌리며 호흡하세요.")

        XCTAssertEqual(session.tick(), .phaseChanged)
        XCTAssertEqual(session.instructionText(), "완료했어요 — 편안하게 남은 휴식을 이어가세요.")
    }

    func testCancelReturnsRunningSessionToReadyAndAllowsRestart() {
        var session = GuidedBreakSession()
        XCTAssertTrue(session.start(availableBreakSeconds: 180))
        advance(&session, ticks: 10)

        session.cancel()
        XCTAssertEqual(session.phase, .ready)
        XCTAssertTrue(session.start(availableBreakSeconds: 170))
        XCTAssertEqual(session.phase, .running(remainingSeconds: 120))
    }

    func testCancelIsNoOpOutsideRunningPhase() {
        var session = GuidedBreakSession()
        session.cancel()
        XCTAssertEqual(session.phase, .ready)

        XCTAssertTrue(session.start(availableBreakSeconds: 123))
        advance(&session, ticks: 120)
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 3))

        session.cancel()
        XCTAssertEqual(session.phase, .completed(remainingDisplaySeconds: 3))
    }

    func testNewSessionDoesNotRestorePreviousPhase() {
        var firstSession = GuidedBreakSession()
        XCTAssertTrue(firstSession.start(availableBreakSeconds: 600))
        advance(&firstSession, ticks: 60)

        let restartedSession = GuidedBreakSession()
        XCTAssertEqual(restartedSession.phase, .ready)
    }

    private func advance(_ session: inout GuidedBreakSession, ticks: Int) {
        for _ in 0..<ticks {
            _ = session.tick()
        }
    }
}
