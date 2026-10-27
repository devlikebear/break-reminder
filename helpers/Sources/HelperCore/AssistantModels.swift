import Foundation

/// Today's briefing and retrospective (`status.assistant`).
public struct AssistantDay: Codable, Equatable {
    public var date = ""
    public var briefing = ""
    public var briefingSource = ""
    public var retro = ""
    public var retroSource = ""
    public var dismissed = false
    public init() {}

    enum CodingKeys: String, CodingKey {
        case date, briefing, retro, dismissed
        case briefingSource = "briefing_source", retroSource = "retro_source"
    }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        date = try c.decodeIfPresent(String.self, forKey: .date) ?? ""
        briefing = try c.decodeIfPresent(String.self, forKey: .briefing) ?? ""
        briefingSource = try c.decodeIfPresent(String.self, forKey: .briefingSource) ?? ""
        retro = try c.decodeIfPresent(String.self, forKey: .retro) ?? ""
        retroSource = try c.decodeIfPresent(String.self, forKey: .retroSource) ?? ""
        dismissed = try c.decodeIfPresent(Bool.self, forKey: .dismissed) ?? false
    }

    /// The card stays until the user closes it.
    public var showsCard: Bool { !dismissed && !(briefing.isEmpty && retro.isEmpty) }
    public var cardText: String { [briefing, retro].filter { !$0.isEmpty }.joined(separator: "\n\n") }
}

/// An action the assistant proposed; shown as a button.
public struct AgentSuggestion: Codable, Equatable, Identifiable {
    public var action: String
    public var minutes = 0
    public var label = ""
    public var mode = ""
    public var reason = ""
    public var id: String { "\(action)-\(minutes)-\(label)-\(mode)" }

    public init(action: String, minutes: Int = 0, label: String = "", mode: String = "", reason: String = "") {
        self.action = action; self.minutes = minutes; self.label = label; self.mode = mode; self.reason = reason
    }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        action = try c.decode(String.self, forKey: .action)
        minutes = try c.decodeIfPresent(Int.self, forKey: .minutes) ?? 0
        label = try c.decodeIfPresent(String.self, forKey: .label) ?? ""
        mode = try c.decodeIfPresent(String.self, forKey: .mode) ?? ""
        reason = try c.decodeIfPresent(String.self, forKey: .reason) ?? ""
    }

    public var params: [String: Any] {
        ["action": action, "minutes": minutes, "label": label, "mode": mode, "reason": reason]
    }

    public var title: String {
        switch action {
        case "start_pomodoro": return L10n.text("Start pomodoro")
        case "start_timer": return L10n.text("Start a {0} min timer", minutes)
        case "pause": return minutes > 0 ? L10n.text("Pause for {0} min", minutes) : L10n.text("Pause")
        case "resume": return L10n.text("Resume")
        default: return action
        }
    }
}

public struct AgentTurn: Codable, Equatable {
    public var role: String
    public var content: String
    public init(role: String, content: String) { self.role = role; self.content = content }
    public var dictionary: [String: Any] { ["role": role, "content": content] }
}

/// The reply to `agent.ask`.
public struct AgentAnswer: Decodable, Equatable {
    public var text = ""
    public var suggestions: [AgentSuggestion] = []
    public var history: [AgentTurn] = []
    enum CodingKeys: String, CodingKey { case text, suggestions, history }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        text = try c.decodeIfPresent(String.self, forKey: .text) ?? ""
        suggestions = try c.decodeIfPresent([AgentSuggestion].self, forKey: .suggestions) ?? []
        history = try c.decodeIfPresent([AgentTurn].self, forKey: .history) ?? []
    }
}

/// `ai.status`: the selected provider, for the settings section.
public struct AIProviderStatus: Decodable, Equatable {
    public var enabled = false
    public var kind = ""
    public var model = ""
    public var ready = false
    public var consented = false
    public var selfExecuting = false
    public var shareEventTitles = false
    public var models: [String] = []
    public var kinds: [String] = []
    public var error: String?
    public init() {}

    enum CodingKeys: String, CodingKey {
        case enabled, kind, model, ready, consented, models, kinds, error
        case selfExecuting = "self_executing", shareEventTitles = "share_event_titles"
    }
    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        enabled = try c.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        kind = try c.decodeIfPresent(String.self, forKey: .kind) ?? ""
        model = try c.decodeIfPresent(String.self, forKey: .model) ?? ""
        ready = try c.decodeIfPresent(Bool.self, forKey: .ready) ?? false
        consented = try c.decodeIfPresent(Bool.self, forKey: .consented) ?? false
        selfExecuting = try c.decodeIfPresent(Bool.self, forKey: .selfExecuting) ?? false
        shareEventTitles = try c.decodeIfPresent(Bool.self, forKey: .shareEventTitles) ?? false
        models = try c.decodeIfPresent([String].self, forKey: .models) ?? []
        kinds = try c.decodeIfPresent([String].self, forKey: .kinds) ?? []
        error = try c.decodeIfPresent(String.self, forKey: .error)
    }

    /// Claude Code signs in on its own; the others need a key.
    public var usesAPIKey: Bool { kind != "claude-code-cli" && !kind.isEmpty }
}

/// Display name for a provider kind.
public func aiProviderName(_ kind: String) -> String {
    switch kind {
    case "claude-code-cli": return L10n.text("Claude Code (your Claude plan)")
    case "anthropic": return "Anthropic API"
    case "openai": return "OpenAI API"
    case "gemini-native": return "Gemini API"
    default: return kind
    }
}
