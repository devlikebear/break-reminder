import SwiftUI
import HelperCore

/// Left-click popover: timer status, controls, pomodoro and countdown cards.
struct PopoverRootView: View {
    @ObservedObject var model: AppModel
    @ObservedObject var calendar: CalendarCollector

    var body: some View {
        ScrollView {
            content
        }
        .frame(width: 320)
        .frame(maxHeight: 640)
    }

    private var content: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let p = model.presentation, let status = model.status {
                StatusCard(model: model, presentation: p)
                let context = status.context ?? WorkContextStatus()
                NowCard(model: model, calendar: calendar, context: context, sessionActive: p.sessionActive)
                PomodoroCard(model: model, presentation: p, status: status)
                CountdownCard(model: model, snapshot: status.timeTools ?? TimeToolsSnapshot())
                AppUsageSection(context: context)
                WorkContextSettings(model: model, calendar: calendar, context: context)
                if let configError = status.configError {
                    Text(L10n.text("Config is invalid, using defaults: {0}", configError))
                        .font(.caption).foregroundStyle(.orange).lineLimit(3)
                }
            } else {
                HStack(spacing: 8) {
                    ProgressView().controlSize(.small)
                    Text(model.engineRunning ? L10n.text("Loading…") : L10n.text("Starting the timer engine…"))
                }
            }
            if let error = model.lastError {
                Text(error).font(.caption).foregroundStyle(.red).textSelection(.enabled).lineLimit(4)
            }
        }
        .padding(14)
    }
}

private typealias Card = PopoverCard

private struct StatusCard: View {
    @ObservedObject var model: AppModel
    let presentation: PopoverPresentation

    private var phaseLabel: String {
        switch presentation.phase {
        case .idle: return L10n.text("Idle")
        case .working: return L10n.text("Working")
        case .onBreak: return L10n.text("On break")
        case .paused: return L10n.text("Paused")
        }
    }

    var body: some View {
        Card {
            HStack {
                Text(phaseLabel).font(.headline)
                Spacer()
                Text(presentation.menuTitle).font(.subheadline.monospacedDigit()).foregroundStyle(.secondary)
            }
            ProgressView(value: presentation.progress)
                .tint(presentation.phase == .onBreak ? .green : .accentColor)
            Text(presentation.statusLine).font(.caption)
            Text(presentation.statsLine).font(.caption).foregroundStyle(.secondary)
            HStack {
                Button(presentation.sessionActive ? L10n.text("Stop Work Session") : L10n.text("Start Work Session")) {
                    model.perform(presentation.sessionActive ? "session.stop" : "session.start")
                }
                if presentation.phase == .paused {
                    Button(L10n.text("Resume")) { model.perform("timer.resume") }
                } else {
                    Menu(L10n.text("Pause")) {
                        Button(L10n.text("Meeting · 30 min")) { model.perform("timer.pause", ["reason": "meeting", "duration_sec": 1800]) }
                        Button(L10n.text("Meeting · 1 hour")) { model.perform("timer.pause", ["reason": "meeting", "duration_sec": 3600]) }
                        Button(L10n.text("Focus · until resumed")) { model.perform("timer.pause", ["reason": "focus"]) }
                        Button(L10n.text("Away · until resumed")) { model.perform("timer.pause", ["reason": "afk"]) }
                    }
                    .fixedSize()
                    .disabled(!presentation.sessionActive)
                }
            }
            .disabled(model.busy)
            HStack {
                if presentation.canSnooze {
                    Button(L10n.text("Snooze 5 min")) { model.perform("timer.snooze", ["seconds": 300]) }
                }
                Button(L10n.text("Reset Timer")) { model.perform("timer.reset") }
            }
            .disabled(model.busy)
        }
    }
}

private struct PomodoroCard: View {
    @ObservedObject var model: AppModel
    let presentation: PopoverPresentation
    let status: EngineStatus

    var body: some View {
        Card {
            Text(presentation.pomodoroLine).font(.subheadline)
            HStack {
                if status.timerMode == "pomodoro" {
                    Button(L10n.text("Classic cycle")) { model.perform("pomodoro.set", ["enabled": false]) }
                } else {
                    Button(L10n.text("Start Pomodoro")) {
                        model.perform("pomodoro.set", ["enabled": true])
                        if !presentation.sessionActive { model.perform("session.start") }
                    }
                }
            }
            .disabled(model.busy || !presentation.canSwitchMode)
        }
    }
}

private struct CountdownCard: View {
    @ObservedObject var model: AppModel
    let snapshot: TimeToolsSnapshot

    var body: some View {
        Card {
            Text(L10n.text("Countdown")).font(.subheadline)
            if let c = snapshot.countdown {
                HStack {
                    Text(c.displayLabel).lineLimit(1)
                    Spacer()
                    Text(c.timeText(nowMS: model.nowUnix * 1000)).font(.body.monospacedDigit())
                }
                HStack {
                    if c.isActive {
                        Button(c.phase == "running" ? L10n.text("Pause") : L10n.text("Resume")) {
                            send(c.phase == "running" ? "pause" : "resume", id: c.id)
                        }
                        Button(L10n.text("Cancel")) { send("cancel", id: c.id) }
                    } else {
                        Button(L10n.text("Restart")) { send("restart", id: c.id) }
                    }
                }
            }
            ForEach(snapshot.unread.reversed()) { event in
                HStack {
                    Text(L10n.text("Completed · {0}", event.displayLabel)).font(.caption).lineLimit(1)
                    Spacer()
                    Button(L10n.text("OK")) { send("acknowledge", id: event.id) }
                }
            }
            HStack {
                ForEach([5, 10, 15, 30], id: \.self) { minutes in
                    Button(L10n.text("{0} min", minutes)) {
                        var params: [String: Any] = ["kind": "start", "duration_sec": minutes * 60, "if_revision": snapshot.revision]
                        if let c = snapshot.countdown, c.isActive { params["replace_id"] = c.id }
                        model.perform("time_tools.command", params)
                    }
                }
            }
        }
        .disabled(model.busy)
    }

    private func send(_ kind: String, id: String) {
        model.perform("time_tools.command", ["kind": kind, "id": id, "if_revision": snapshot.revision])
    }
}
