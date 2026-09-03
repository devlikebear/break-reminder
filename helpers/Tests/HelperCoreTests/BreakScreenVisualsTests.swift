import XCTest
@testable import HelperCore

final class BreakScreenVisualsTests: XCTestCase {
    func testMascotPresentationUsesAQuietLowMotionContract() {
        XCTAssertEqual(BreakScreenVisuals.mascotResourceName, "BreakMascot")
        XCTAssertGreaterThanOrEqual(BreakScreenVisuals.breathingDuration, 4.0)
        XCTAssertLessThanOrEqual(BreakScreenVisuals.breathingDuration, 6.0)
        XCTAssertGreaterThan(BreakScreenVisuals.breathingScale, 0)
        XCTAssertLessThanOrEqual(BreakScreenVisuals.breathingScale, 0.05)
        XCTAssertLessThanOrEqual(BreakScreenVisuals.entryFadeDuration, 0.5)
    }
}
