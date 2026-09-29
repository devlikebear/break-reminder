import XCTest
@testable import HelperCore

final class PomodoroControlsTests: XCTestCase {
    func testStartSelectsModeAndStartsSession() {
        XCTAssertEqual(pomodoroStartCommands(state: AppState(), config: AppConfig()), [["pomodoro", "on"], ["start"]])
        var cfg = AppConfig(); cfg.timerMode = "pomodoro"
        XCTAssertEqual(pomodoroStartCommands(state: AppState(), config: cfg), [["start"]])
        var state = AppState(); state.sessionState = "active"
        XCTAssertEqual(pomodoroStartCommands(state: state, config: cfg), [])
        state.paused = true
        XCTAssertEqual(pomodoroStartCommands(state: state, config: cfg), [])
    }
    func testRoundAndLongBreakAreUnambiguous() {
        var cfg = AppConfig(); cfg.timerMode = "pomodoro"
        var state = AppState(); state.pomodoroCount = 3; state.todayPomodoros = 7
        XCTAssertEqual(pomodoroStatus(state: state, config: cfg), "집중 4/4 · 오늘 완료 7회")
        state.mode = "break"; state.pomodoroCount = 4
        XCTAssertTrue(pomodoroStatus(state: state, config: cfg).hasPrefix("긴 휴식"))
    }
    func testEditingDoesNotTriggerAppShortcuts() {
        XCTAssertFalse(shouldHandlePlainShortcut(isEditing: true, hasModifiers: false))
        XCTAssertFalse(shouldHandlePlainShortcut(isEditing: false, hasModifiers: true))
        XCTAssertTrue(shouldHandlePlainShortcut(isEditing: false, hasModifiers: false))
    }
}

private actor PomodoroMockClient: CLICommandRunning {
    var calls: [[String]] = []
    let failureIndex: Int
    init(failureIndex: Int) { self.failureIndex = failureIndex }
    func run(arguments: [String]) async -> CommandResult {
        calls.append(arguments)
        try? await Task.sleep(nanoseconds: 20_000_000)
        return CommandResult(exitCode: calls.count == failureIndex ? 1 : 0, stderr: "test failure")
    }
    func recorded() -> [[String]] { calls }
}

extension PomodoroControlsTests {
    @MainActor
    func testPartialSuccessStopsSequenceAndExplainsSavedMode() async {
        let actions = PomodoroActions()
        let client = PomodoroMockClient(failureIndex: 2)
        await actions.perform(commands: [["pomodoro", "on"], ["start"], ["unexpected"]], client: client)
        let calls = await client.recorded()
        XCTAssertEqual(calls, [["pomodoro", "on"], ["start"]])
        XCTAssertTrue(actions.message?.contains("설정은 저장됐지만") == true)
        XCTAssertFalse(actions.isBusy)
    }
    @MainActor
    func testDuplicateStartWhileBusyDoesNotRunTwice() async {
        let actions = PomodoroActions()
        let client = PomodoroMockClient(failureIndex: 0)
        let first = Task { await actions.perform(commands: [["start"]], client: client) }
        while !actions.isBusy { await Task.yield() }
        await actions.perform(commands: [["start"]], client: client)
        await first.value
        let calls = await client.recorded()
        XCTAssertEqual(calls, [["start"]])
    }
}
