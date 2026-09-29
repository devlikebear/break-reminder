import SwiftUI
import HelperCore

struct CountdownView: View {
    @StateObject private var model = TimeToolsViewModel()
    @State private var label = ""
    @State private var hours = 0
    @State private var minutes = 5
    @State private var seconds = 0
    @State private var replaceID: String?
    @State private var confirmReplace = false
    @State private var confirmCancel = false
    @State private var cancelID = ""
    private var totalSeconds: Int { hours * 3600 + minutes * 60 + seconds }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L10n.text("Countdown")).font(.headline)
            Text(L10n.text("Runs independently of work and breaks, even after closing this window.")).font(.caption).foregroundStyle(.secondary)
            if !model.workerRunning {
                Label(L10n.text("Background timers need setup or are stopped."), systemImage: "exclamationmark.triangle")
                Button(L10n.text("Set up / repair time tools")) { model.setupService() }.disabled(model.isBusy)
                Text(L10n.text("Installs and starts the background services.")).font(.caption)
            } else if !model.runtime.notificationAvailable {
                Text(L10n.text("System notifications are unavailable. Check completion in the app; sound is not guaranteed.")).font(.caption).foregroundStyle(.orange)
            }
            if let error = model.readError { Text(error).foregroundStyle(.red).font(.caption) }
            if let error = model.commandError { Text(error).foregroundStyle(.red).font(.caption).textSelection(.enabled) }
            if let countdown = model.snapshot.countdown {
                VStack(alignment: .leading, spacing: 8) {
                    Text(countdown.displayLabel).font(.headline).lineLimit(2)
                    Text(countdown.timeText(nowMS: model.nowMS)).font(.system(size: 32, weight: .medium, design: .monospaced))
                    Text(phaseLabel(countdown.phase)).font(.caption)
                    HStack {
                        if countdown.isActive {
                            Button(countdown.phase == "running" ? L10n.text("Pause") : L10n.text("Resume")) {
                                model.send(["timer", countdown.phase == "running" ? "pause" : "resume", "--id", countdown.id])
                            }.disabled(model.isBusy || (countdown.phase == "paused" && !model.workerRunning))
                            Button(L10n.text("Cancel")) { cancelID = countdown.id; confirmCancel = true }.disabled(model.isBusy)
                        } else {
                            Button(L10n.text("Restart")) { model.send(["timer", "restart", "--id", countdown.id]) }.disabled(model.isBusy || !model.workerRunning)
                        }
                    }
                }.padding(10).frame(maxWidth: .infinity, alignment: .leading).background(.quaternary, in: RoundedRectangle(cornerRadius: 8))
            }
            ForEach(model.snapshot.unread.reversed()) { event in
                VStack(alignment: .leading, spacing: 6) {
                    Text(L10n.text("Completed · {0}", event.displayLabel)).font(.headline)
                    if event.deliveryState == "silent" { Text(L10n.text("Completed while you were away.")).font(.caption) }
                    if event.deliveryState == "failed" || event.deliveryState == "unknown" {
                        Text(L10n.text("Could not confirm notification delivery.")).font(.caption)
                        Button(L10n.text("Retry notification")) { model.send(["notify-again", "--event-id", event.id]) }.disabled(!model.workerRunning || model.isBusy)
                    }
                    Button(L10n.text("OK")) { model.send(["acknowledge", "--event-id", event.id]) }.disabled(model.isBusy)
                }.padding(8).frame(maxWidth: .infinity, alignment: .leading).background(.orange.opacity(0.12), in: RoundedRectangle(cornerRadius: 8))
            }
            Divider()
            TextField(L10n.text("Name (optional, up to 80 characters)"), text: $label).textFieldStyle(.roundedBorder)
            HStack {
                ForEach([5, 10, 15, 30], id: \.self) { value in
                    Button(L10n.text("{0} min", value)) { hours = 0; minutes = value; seconds = 0 }
                }
            }
            HStack {
                timeField(L10n.text("h"), value: $hours)
                timeField(L10n.text("min"), value: $minutes)
                timeField(L10n.text("s"), value: $seconds)
            }
            Button(model.snapshot.countdown?.isActive == true ? L10n.text("Replace with a new timer…") : L10n.text("Start")) {
                if let current = model.snapshot.countdown, current.isActive { replaceID = current.id; confirmReplace = true }
                else { model.startTimer(seconds: totalSeconds, label: label) }
            }
            .disabled(!model.workerRunning || model.isBusy || model.readError != nil || totalSeconds < 1 || totalSeconds > 86400 || hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59)
            if !model.snapshot.recent.isEmpty {
                Text(L10n.text("Recent timers")).font(.caption).foregroundStyle(.secondary)
                ForEach(Array(model.snapshot.recent.enumerated()), id: \.offset) { _, recent in
                    Button("\(recent.displayLabel) · \(timeToolsDuration(recent.durationMS))") {
                        label = recent.label
                        let duration = Int(recent.durationMS / 1000)
                        hours = duration / 3600; minutes = duration / 60 % 60; seconds = duration % 60
                    }.lineLimit(1)
                }
            }
        }
        .onAppear { model.startRefreshing() }
        .onDisappear { model.stopRefreshing() }
        .confirmationDialog(L10n.text("Cancel the active timer and start a new one?"), isPresented: $confirmReplace) {
            Button(L10n.text("Replace"), role: .destructive) { model.startTimer(seconds: totalSeconds, label: label, replaceID: replaceID) }
        }
        .confirmationDialog(L10n.text("Cancel this timer?"), isPresented: $confirmCancel) {
            Button(L10n.text("Cancel timer"), role: .destructive) { model.send(["timer", "cancel", "--id", cancelID]) }
        }
    }
    private func timeField(_ title: String, value: Binding<Int>) -> some View {
        HStack(spacing: 3) {
            TextField(title, value: value, formatter: integerFormatter).textFieldStyle(.roundedBorder).frame(minWidth: 40)
            Text(title).font(.caption)
        }
    }
    private var integerFormatter: NumberFormatter { let f = NumberFormatter(); f.numberStyle = .none; f.allowsFloats = false; f.minimum = 0; f.maximum = 86400; return f }
    private func phaseLabel(_ phase: String) -> String {
        switch phase { case "running": return L10n.text("Running"); case L10n.text("paused"): return L10n.text("Paused"); case "completed": return L10n.text("Completed"); default: return L10n.text("Canceled") }
    }
}
