import HelperCore
import SwiftUI

/// Today's briefing/retro, until the user closes it.
struct BriefingCard: View {
    @ObservedObject var model: AppModel
    let day: AssistantDay

    var body: some View {
        PopoverCard {
            HStack(alignment: .top) {
                Text(day.retro.isEmpty ? L10n.text("Today's briefing") : L10n.text("Today's retrospective")).font(.subheadline)
                Spacer()
                Button {
                    model.perform("assistant.dismiss")
                } label: {
                    Image(systemName: "xmark").font(.caption)
                }
                .buttonStyle(.borderless)
                .accessibilityLabel(L10n.text("Close"))
            }
            Text(day.cardText).font(.callout).fixedSize(horizontal: false, vertical: true).textSelection(.enabled)
        }
    }
}

/// The question box: streamed answers and suggestion buttons.
struct AskSection: View {
    @ObservedObject var assistant: AssistantModel

    var body: some View {
        PopoverCard {
            Text(L10n.text("Ask")).font(.subheadline)
            ForEach(Array(assistant.turns.enumerated()), id: \.offset) { _, turn in
                Text(turn.content)
                    .font(.callout)
                    .foregroundStyle(turn.role == "user" ? .secondary : .primary)
                    .frame(maxWidth: .infinity, alignment: turn.role == "user" ? .trailing : .leading)
                    .fixedSize(horizontal: false, vertical: true)
                    .textSelection(.enabled)
            }
            if assistant.asking {
                HStack(alignment: .top, spacing: 6) {
                    ProgressView().controlSize(.small)
                    Text(assistant.streaming.isEmpty ? L10n.text("Thinking…") : assistant.streaming)
                        .font(.callout).fixedSize(horizontal: false, vertical: true)
                }
            }
            ForEach(assistant.suggestions) { s in
                VStack(alignment: .leading, spacing: 2) {
                    Button(s.title) { assistant.run(s) }
                    if !s.reason.isEmpty { Text(s.reason).font(.caption2).foregroundStyle(.secondary) }
                }
            }
            if let error = assistant.error {
                Text(error).font(.caption).foregroundStyle(.red).fixedSize(horizontal: false, vertical: true)
            }
            HStack {
                TextField(L10n.text("e.g. What should I do before my next meeting?"), text: $assistant.question)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit { assistant.ask() }
                Button(L10n.text("Ask")) { assistant.ask() }
                    .disabled(assistant.asking || assistant.question.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            Toggle(L10n.text("Let the assistant change the timer"), isOn: $assistant.allowActions)
                .font(.caption)
        }
    }
}

/// Provider, key, consent and briefing settings.
struct AISettings: View {
    @ObservedObject var assistant: AssistantModel
    @State private var expanded = false

    var body: some View {
        let p = assistant.provider
        DisclosureGroup(isExpanded: $expanded) {
            VStack(alignment: .leading, spacing: 8) {
                Toggle(L10n.text("AI assistant"), isOn: Binding(get: { p.enabled }, set: { assistant.setting(["ai_enabled": $0]) }))
                Picker(L10n.text("Provider"), selection: Binding(get: { p.kind }, set: { assistant.setting(["ai_provider": $0]) })) {
                    ForEach(p.kinds, id: \.self) { Text(aiProviderName($0)).tag($0) }
                }
                if !p.models.isEmpty {
                    Picker(L10n.text("Model"), selection: Binding(get: { p.model }, set: { assistant.setting(["ai_model": $0]) })) {
                        Text(L10n.text("Default")).tag("")
                        ForEach(p.models, id: \.self) { Text($0).tag($0) }
                    }
                }
                if p.usesAPIKey {
                    HStack {
                        SecureField(L10n.text("API key (stored in Keychain)"), text: $assistant.apiKeyDraft)
                            .textFieldStyle(.roundedBorder)
                        Button(L10n.text("Save")) { assistant.saveKey() }
                            .disabled(assistant.apiKeyDraft.isEmpty)
                    }
                }
                Toggle(L10n.text("Allow {0} to receive my calendar and app usage", aiProviderName(p.kind)),
                       isOn: Binding(get: { p.consented }, set: { assistant.consent($0) }))
                    .fixedSize(horizontal: false, vertical: true)
                Toggle(L10n.text("Include event titles"), isOn: Binding(get: { p.shareEventTitles }, set: { assistant.setting(["share_event_titles": $0]) }))
                    .disabled(!p.consented)
                Text(L10n.text("Without consent the briefing and retrospective use built-in sentences and the question box is off."))
                    .font(.caption2).foregroundStyle(.secondary).fixedSize(horizontal: false, vertical: true)
                HStack {
                    Button(L10n.text("Test connection")) { assistant.test() }
                    if let result = assistant.testResult { Text(result).font(.caption).lineLimit(2) }
                }
                Text(p.ready ? L10n.text("Ready") : (p.error ?? L10n.text("Not ready")))
                    .font(.caption2).foregroundStyle(p.ready ? Color.secondary : Color.orange)
                    .fixedSize(horizontal: false, vertical: true)
            }
        } label: {
            Text(L10n.text("AI assistant")).font(.subheadline)
        }
        .onAppear { assistant.refreshProvider() }
    }
}
