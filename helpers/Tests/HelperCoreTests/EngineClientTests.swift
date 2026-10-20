import XCTest
@testable import HelperCore

/// A stand-in engine: answers every request with its method name, announces
/// an event first, and exits on "crash".
private let fakeEngine = """
import json, sys
print(json.dumps({"jsonrpc": "2.0", "method": "event", "params": {"type": "session_start", "title": "hi"}}), flush=True)
for line in sys.stdin:
    req = json.loads(line)
    if req["method"] == "crash":
        sys.exit(3)
    if req["method"] == "fail":
        print(json.dumps({"jsonrpc": "2.0", "id": req["id"], "error": {"code": -32001, "message": "nope"}}), flush=True)
        continue
    print(json.dumps({"jsonrpc": "2.0", "id": req["id"], "result": {"method": req["method"], "params": req.get("params")}}), flush=True)
"""

final class EngineClientTests: XCTestCase {
    private func makeClient() -> EngineClient {
        EngineClient(configuration: .init(
            executableURL: URL(fileURLWithPath: "/usr/bin/env"),
            arguments: ["python3", "-u", "-c", fakeEngine],
            environment: ProcessInfo.processInfo.environment,
            restartDelay: { _ in 0.05 }
        ))
    }

    func testCallRoundTripAndEvents() async throws {
        let client = makeClient()
        let gotEvent = expectation(description: "event")
        client.onEvent = { event in if event.type == "session_start" { gotEvent.fulfill() } }
        client.start()
        defer { client.stop() }

        let data = try await client.call("timer.pause", params: ["reason": "focus"])
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        XCTAssertEqual(object?["method"] as? String, "timer.pause")
        XCTAssertEqual((object?["params"] as? [String: Any])?["reason"] as? String, "focus")

        do {
            _ = try await client.call("fail")
            XCTFail("expected RPC error")
        } catch let error as EngineRPCError {
            XCTAssertEqual(error.code, -32001)
        }
        await fulfillment(of: [gotEvent], timeout: 5)
    }

    func testCrashFailsPendingCallAndRestarts() async throws {
        let client = makeClient()
        client.start()
        defer { client.stop() }
        _ = try await client.call("engine.ping")
        do {
            _ = try await client.call("crash", timeout: 5)
            XCTFail("call to a crashing engine should fail")
        } catch {}
        // The supervisor relaunches the engine and calls work again.
        var answered = false
        for _ in 0..<50 where !answered {
            if let _ = try? await client.call("engine.ping", timeout: 1) { answered = true } else {
                try await Task.sleep(nanoseconds: 100_000_000)
            }
        }
        XCTAssertTrue(answered)
        XCTAssertGreaterThanOrEqual(client.launchCount, 2)
    }

    func testCallBeforeStartFails() async {
        let client = makeClient()
        do {
            _ = try await client.call("engine.ping", timeout: 0.2)
            XCTFail("expected failure")
        } catch {}
    }

    func testStopDoesNotRestart() async throws {
        let client = makeClient()
        client.start()
        _ = try await client.call("engine.ping")
        client.stop()
        try await Task.sleep(nanoseconds: 300_000_000)
        XCTAssertEqual(client.launchCount, 1)
        XCTAssertFalse(client.isRunning)
    }
}
