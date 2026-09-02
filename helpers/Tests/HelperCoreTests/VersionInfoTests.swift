import XCTest
@testable import HelperCore

final class VersionInfoTests: XCTestCase {
    func testStandardOutput() {
        XCTAssertEqual(parseVersionOutput("break-reminder 0.13.0"), "0.13.0")
    }

    func testTrailingNewline() {
        XCTAssertEqual(parseVersionOutput("break-reminder 0.13.0\n"), "0.13.0")
    }

    func testDevBuild() {
        XCTAssertEqual(parseVersionOutput("break-reminder dev"), "dev")
    }

    func testBareVersion() {
        XCTAssertEqual(parseVersionOutput("0.13.0"), "0.13.0")
    }

    func testExtraWhitespace() {
        XCTAssertEqual(parseVersionOutput("  break-reminder   0.13.0  "), "0.13.0")
    }

    /// Leading blank lines can appear when the CLI emits a warning banner first.
    func testSkipsLeadingBlankLines() {
        XCTAssertEqual(parseVersionOutput("\n\nbreak-reminder 0.13.0\n"), "0.13.0")
    }

    /// Only the first non-empty line matters; later lines are ignored.
    func testIgnoresTrailingLines() {
        XCTAssertEqual(parseVersionOutput("break-reminder 0.13.0\nsomething else\n"), "0.13.0")
    }

    func testEmptyInput() {
        XCTAssertEqual(parseVersionOutput(""), AboutInfo.unknownVersion)
    }

    func testWhitespaceOnlyInput() {
        XCTAssertEqual(parseVersionOutput("  \n\t\n "), AboutInfo.unknownVersion)
    }
}
