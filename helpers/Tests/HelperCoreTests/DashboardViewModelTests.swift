import XCTest
@testable import HelperCore

final class DashboardViewModelTests: XCTestCase {
    func testWorkProgressCalculation() {
        let state = AppState()
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let sp = workProgress(state: state, config: config, now: now)
        XCTAssertGreaterThanOrEqual(sp.progress, 0.0)
        XCTAssertLessThanOrEqual(sp.progress, 1.0)
    }

    func testBreakProgressCalculation() {
        var state = AppState()
        state.mode = "break"
        state.breakStart = Int64(Date().timeIntervalSince1970) - 60
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let sp = breakProgress(state: state, config: config, now: now)
        XCTAssertGreaterThan(sp.elapsedSec, 0)
    }

    func testLiveDailyTotalsDefaultState() {
        let state = AppState()
        let config = AppConfig()
        let now = Int64(Date().timeIntervalSince1970)

        let totals = liveDailyTotals(state: state, config: config, now: now)
        XCTAssertGreaterThanOrEqual(totals.workSeconds, 0)
        XCTAssertGreaterThanOrEqual(totals.breakSeconds, 0)
    }
}
