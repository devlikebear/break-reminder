import XCTest
@testable import HelperCore

final class CLICommandClientTests: XCTestCase {
    func testSuccessAndFailureOutput() async {
        let client = CLICommandClient(executablePath: "/bin/sh")
        let good = await client.run(arguments: ["-c", "printf success"])
        XCTAssertTrue(good.succeeded)
        XCTAssertEqual(good.stdout, "success")
        let bad = await client.run(arguments: ["-c", "printf failure >&2; exit 7"])
        XCTAssertFalse(bad.succeeded)
        XCTAssertEqual(bad.exitCode, 7)
        XCTAssertEqual(bad.stderr, "failure")
    }
    func testBothPipesAreDrainedWithoutDeadlock() async {
        let client = CLICommandClient(executablePath: "/bin/sh", timeout: 5)
        let result = await client.run(arguments: ["-c", "i=0; while [ $i -lt 10000 ]; do printf 'stdout test line!\n'; printf 'stderr test line!\n' >&2; i=$((i+1)); done"])
        XCTAssertTrue(result.succeeded)
        XCTAssertGreaterThan(result.stdout.count, 65536)
        XCTAssertGreaterThan(result.stderr.count, 65536)
    }
    func testMissingBinaryAndTimeout() async {
        let missing = await CLICommandClient(executablePath: "/missing/cli").run(arguments: [])
        XCTAssertFalse(missing.succeeded)
        let timeout = await CLICommandClient(executablePath: "/bin/sleep", timeout: 0.05).run(arguments: ["2"])
        XCTAssertTrue(timeout.timedOut)
        XCTAssertFalse(timeout.succeeded)
    }
}
