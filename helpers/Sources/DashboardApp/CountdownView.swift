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
            Text("일반 타이머").font(.headline)
            Text("업무·휴식과 독립적으로 실행됩니다. 창을 닫아도 계속됩니다.").font(.caption).foregroundStyle(.secondary)
            if !model.workerRunning {
                Label("백그라운드 타이머 설정이 필요하거나 중지됐습니다.", systemImage: "exclamationmark.triangle")
                Button("시간 도구 설정 / 복구") { model.setupService() }.disabled(model.isBusy)
                Text("백그라운드 서비스를 설치·시작합니다.").font(.caption)
            } else if !model.runtime.notificationAvailable {
                Text("시스템 알림을 사용할 수 없음 — 완료는 앱에서 확인하세요. 소리는 보장되지 않습니다.").font(.caption).foregroundStyle(.orange)
            }
            if let error = model.readError { Text(error).foregroundStyle(.red).font(.caption) }
            if let error = model.commandError { Text(error).foregroundStyle(.red).font(.caption).textSelection(.enabled) }
            if let countdown = model.snapshot.countdown {
                VStack(alignment: .leading, spacing: 8) {
                    Text(countdown.label).font(.headline).lineLimit(2)
                    Text(countdown.timeText(nowMS: model.nowMS)).font(.system(size: 32, weight: .medium, design: .monospaced))
                    Text(phaseLabel(countdown.phase)).font(.caption)
                    HStack {
                        if countdown.isActive {
                            Button(countdown.phase == "running" ? "일시정지" : "재개") {
                                model.send(["timer", countdown.phase == "running" ? "pause" : "resume", "--id", countdown.id])
                            }.disabled(model.isBusy || (countdown.phase == "paused" && !model.workerRunning))
                            Button("취소") { cancelID = countdown.id; confirmCancel = true }.disabled(model.isBusy)
                        } else {
                            Button("다시 시작") { model.send(["timer", "restart", "--id", countdown.id]) }.disabled(model.isBusy || !model.workerRunning)
                        }
                    }
                }.padding(10).frame(maxWidth: .infinity, alignment: .leading).background(.quaternary, in: RoundedRectangle(cornerRadius: 8))
            }
            ForEach(model.snapshot.unread.reversed()) { event in
                VStack(alignment: .leading, spacing: 6) {
                    Text("완료 · \(event.label)").font(.headline)
                    if event.deliveryState == "silent" { Text("자리를 비운 동안 완료됐습니다.").font(.caption) }
                    if event.deliveryState == "failed" || event.deliveryState == "unknown" {
                        Text("시스템 알림 전달을 확인하지 못했습니다.").font(.caption)
                        Button("알림 다시 보내기") { model.send(["notify-again", "--event-id", event.id]) }.disabled(!model.workerRunning || model.isBusy)
                    }
                    Button("확인") { model.send(["acknowledge", "--event-id", event.id]) }.disabled(model.isBusy)
                }.padding(8).frame(maxWidth: .infinity, alignment: .leading).background(.orange.opacity(0.12), in: RoundedRectangle(cornerRadius: 8))
            }
            Divider()
            TextField("이름 (선택, 80자 이내)", text: $label).textFieldStyle(.roundedBorder)
            HStack {
                ForEach([5, 10, 15, 30], id: \.self) { value in
                    Button("\(value)분") { hours = 0; minutes = value; seconds = 0 }
                }
            }
            HStack {
                timeField("시", value: $hours)
                timeField("분", value: $minutes)
                timeField("초", value: $seconds)
            }
            Button(model.snapshot.countdown?.isActive == true ? "새 타이머로 교체…" : "시작") {
                if let current = model.snapshot.countdown, current.isActive { replaceID = current.id; confirmReplace = true }
                else { model.startTimer(seconds: totalSeconds, label: label) }
            }
            .disabled(!model.workerRunning || model.isBusy || model.readError != nil || totalSeconds < 1 || totalSeconds > 86400 || hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59)
            if !model.snapshot.recent.isEmpty {
                Text("최근 사용").font(.caption).foregroundStyle(.secondary)
                ForEach(Array(model.snapshot.recent.enumerated()), id: \.offset) { _, recent in
                    Button("\(recent.label) · \(timeToolsDuration(recent.durationMS))") {
                        label = recent.label
                        let duration = Int(recent.durationMS / 1000)
                        hours = duration / 3600; minutes = duration / 60 % 60; seconds = duration % 60
                    }.lineLimit(1)
                }
            }
        }
        .onAppear { model.startRefreshing() }
        .onDisappear { model.stopRefreshing() }
        .confirmationDialog("실행 중인 타이머를 취소하고 새로 시작할까요?", isPresented: $confirmReplace) {
            Button("교체", role: .destructive) { model.startTimer(seconds: totalSeconds, label: label, replaceID: replaceID) }
        }
        .confirmationDialog("타이머를 취소할까요?", isPresented: $confirmCancel) {
            Button("타이머 취소", role: .destructive) { model.send(["timer", "cancel", "--id", cancelID]) }
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
        switch phase { case "running": return "실행 중"; case "paused": return "일시정지"; case "completed": return "완료"; default: return "취소됨" }
    }
}
