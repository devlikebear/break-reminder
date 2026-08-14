import AppKit
import XCTest
@testable import HelperCore

final class HamsterVectorArtworkTests: XCTestCase {
    func testProvidesCompleteAnimationFrameSet() {
        XCTAssertEqual(HamsterVectorArtwork.running.count, MenuBarAnimationFrameCounts.running)
        XCTAssertEqual(HamsterVectorArtwork.resting.count, MenuBarAnimationFrameCounts.resting)
        XCTAssertFalse(HamsterVectorArtwork.sleeping.isEmpty)
    }

    func testFramesDecodeAsConsistentSVGImages() {
        let frames = HamsterVectorArtwork.running
            + HamsterVectorArtwork.resting
            + [HamsterVectorArtwork.sleeping]

        for svg in frames {
            XCTAssertTrue(svg.contains("viewBox=\"0 0 100 72\""))
            XCTAssertNotNil(NSImage(data: Data(svg.utf8)))
        }
    }

    func testFramesArePureEmbeddedVectorsWithoutRasterImages() {
        let frames = HamsterVectorArtwork.running
            + HamsterVectorArtwork.resting
            + [HamsterVectorArtwork.sleeping]

        for svg in frames {
            XCTAssertTrue(svg.hasPrefix("<svg"))
            XCTAssertTrue(svg.contains("viewBox="))
            XCTAssertTrue(svg.contains("<path"))
            XCTAssertFalse(svg.contains("<image"))
            XCTAssertFalse(svg.contains("data:image"))
        }
    }
}
