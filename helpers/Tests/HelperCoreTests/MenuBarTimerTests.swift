import Foundation
import XCTest
@testable import HelperCore

final class MenuBarTimerTests: XCTestCase {
    func testRepeatingTimerFiresDuringEventTrackingRunLoopMode() {
        var didFire = false
        let timer = scheduleMenuBarTimer(interval: 0.01, runLoop: .current) {
            didFire = true
        }
        defer { timer.invalidate() }

        let deadline = Date().addingTimeInterval(0.25)
        while !didFire && Date() < deadline {
            _ = RunLoop.current.run(
                mode: RunLoop.Mode("NSEventTrackingRunLoopMode"),
                before: Date().addingTimeInterval(0.01)
            )
        }

        XCTAssertTrue(didFire, "A common-mode timer should fire during event tracking")
    }
}
