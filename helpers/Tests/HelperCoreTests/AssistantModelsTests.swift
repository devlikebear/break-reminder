import XCTest
@testable import HelperCore

final class AssistantModelsTests: XCTestCase {
    func testAgentDeltaIsClassified() throws {
        switch try EngineInbound.parse(Data(#"{"jsonrpc":"2.0","method":"agent.delta","params":{"request_id":"r1","text":"Two "}}"#.utf8)) {
        case let .agentDelta(id, text):
            XCTAssertEqual(id, "r1"); XCTAssertEqual(text, "Two ")
        default: XCTFail("expected agent delta")
        }
    }

    func testAnswerAndProviderStatusDecode() throws {
        let answer = try JSONDecoder().decode(AgentAnswer.self, from: Data("""
        {"text":"Start now.","suggestions":[{"action":"start_timer","minutes":15,"label":"stretch","reason":"gap"}],
         "history":[{"role":"user","content":"q"},{"role":"assistant","content":"Start now."}]}
        """.utf8))
        XCTAssertEqual(answer.suggestions.first?.action, "start_timer")
        XCTAssertEqual(answer.suggestions.first?.minutes, 15)
        XCTAssertEqual(answer.history.count, 2)
        XCTAssertEqual(answer.suggestions.first?.params["label"] as? String, "stretch")

        let st = try JSONDecoder().decode(AIProviderStatus.self, from: Data("""
        {"enabled":true,"kind":"claude-code-cli","model":"","ready":true,"consented":false,"self_executing":true,
         "share_event_titles":false,"models":["haiku","sonnet"],"kinds":["claude-code-cli","anthropic"]}
        """.utf8))
        XCTAssertTrue(st.ready); XCTAssertFalse(st.consented); XCTAssertFalse(st.usesAPIKey)
        XCTAssertEqual(st.kinds.count, 2)
        let empty = try JSONDecoder().decode(AIProviderStatus.self, from: Data("{}".utf8))
        XCTAssertFalse(empty.ready)
    }

    func testAssistantDayAndCard() throws {
        let day = try JSONDecoder().decode(AssistantDay.self, from: Data(#"{"date":"2026-10-27","briefing":"Morning.","briefing_source":"ai","retro":"Done.","dismissed":false}"#.utf8))
        XCTAssertTrue(day.showsCard)
        XCTAssertEqual(day.cardText, "Morning.\n\nDone.")
        var dismissed = day; dismissed.dismissed = true
        XCTAssertFalse(dismissed.showsCard)
        XCTAssertFalse(AssistantDay().showsCard)
    }

    func testSuggestionTitles() {
        setenv("BREAK_REMINDER_LANGUAGE", "en", 1)
        XCTAssertEqual(AgentSuggestion(action: "start_timer", minutes: 15, label: "", mode: "", reason: "").title, "Start a 15 min timer")
        XCTAssertEqual(AgentSuggestion(action: "start_pomodoro").title, "Start pomodoro")
        XCTAssertEqual(AgentSuggestion(action: "pause", minutes: 30, label: "", mode: "meeting", reason: "").title, "Pause for 30 min")
    }
}
