import Foundation
import HelperCore

/// The popover's conversation and the AI settings. A conversation lasts while
/// the popover is open; closing it starts fresh.
@MainActor
final class AssistantModel: ObservableObject {
    @Published var question = ""
    @Published var allowActions = false
    @Published private(set) var turns: [AgentTurn] = []
    @Published private(set) var streaming = ""
    @Published private(set) var suggestions: [AgentSuggestion] = []
    @Published private(set) var asking = false
    @Published private(set) var error: String?
    @Published private(set) var provider = AIProviderStatus()
    @Published private(set) var testResult: String?
    @Published var apiKeyDraft = ""

    private let model: AppModel
    private var requestID = ""
    private var counter = 0

    init(model: AppModel) {
        self.model = model
        model.client?.onAgentDelta = { [weak self] id, text in
            guard let self, id == self.requestID else { return }
            self.streaming += text
        }
    }

    func reset() {
        guard !asking else { return }
        turns = []; streaming = ""; suggestions = []; error = nil; question = ""
    }

    func ask() {
        let q = question.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty, !asking, let client = model.client else { return }
        counter += 1
        requestID = "ask-\(counter)"
        asking = true; error = nil; streaming = ""; suggestions = []
        let history = turns.map(\.dictionary)
        turns.append(AgentTurn(role: "user", content: q))
        question = ""
        Task {
            defer { asking = false }
            do {
                let answer = try await client.call("agent.ask", params: [
                    "request_id": requestID, "question": q, "history": history, "allow_actions": allowActions,
                ], as: AgentAnswer.self, timeout: 240)
                turns = answer.history.isEmpty ? turns + [AgentTurn(role: "assistant", content: answer.text)] : answer.history
                suggestions = answer.suggestions
                streaming = ""
            } catch {
                self.error = error.localizedDescription
                streaming = ""
            }
        }
    }

    func run(_ suggestion: AgentSuggestion) {
        suggestions.removeAll { $0 == suggestion }
        model.perform("suggestion.run", suggestion.params)
    }

    func refreshProvider() {
        guard let client = model.client else { return }
        Task { if let st = try? await client.call("ai.status", as: AIProviderStatus.self) { provider = st } }
    }

    func setting(_ params: [String: Any]) { update("ai.settings.set", params) }
    func consent(_ on: Bool) { update("ai.consent.set", ["provider": provider.kind, "consent": on]) }

    func saveKey() {
        let key = apiKeyDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !key.isEmpty else { return }
        apiKeyDraft = ""
        update("ai.key.set", ["provider": provider.kind, "key": key])
    }

    func test() {
        guard let client = model.client else { return }
        testResult = L10n.text("Testing…")
        Task {
            do {
                _ = try await client.call("ai.test", timeout: 120)
                testResult = L10n.text("Connected.")
            } catch {
                testResult = error.localizedDescription
            }
        }
    }

    private func update(_ method: String, _ params: [String: Any]) {
        guard let client = model.client else { return }
        Task {
            do {
                provider = try await client.call(method, params: params, as: AIProviderStatus.self)
                error = nil
            } catch {
                self.error = error.localizedDescription
            }
        }
    }
}
