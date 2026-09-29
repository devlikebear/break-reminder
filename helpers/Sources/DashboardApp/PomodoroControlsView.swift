import SwiftUI
import HelperCore

struct PomodoroControlsView: View {
    @ObservedObject var vm: DashboardViewModel
    @StateObject private var actions = PomodoroActions()
    @State private var pendingCommands: [[String]] = []
    @State private var confirmSwitch = false
    @State private var editing = false
    @State private var work = 25
    @State private var shortBreak = 5
    @State private var longBreak = 15
    @State private var every = 4

    private var canSwitch: Bool { !vm.state.paused && vm.state.mode == "work" && !actions.isBusy }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(pomodoroStatus(state: vm.state, config: vm.config)).font(.headline)
            Text("활동 감지 기준 · 오래 자리를 비우면 집중 구간이 초기화됩니다.")
                .font(.caption).foregroundStyle(.secondary)
            HStack {
                Button(vm.config.pomodoroEnabled && vm.state.isSessionActive ? "포모도로 실행 중" : "포모도로 시작") {
                    request(pomodoroStartCommands(state: vm.state, config: vm.config))
                }
                .disabled(!canSwitch || (vm.config.pomodoroEnabled && vm.state.isSessionActive))
                if vm.config.pomodoroEnabled {
                    Button("기본 주기로") { request([["pomodoro", "off"]]) }.disabled(!canSwitch)
                }
            }
            if vm.state.paused || vm.state.mode == "break" {
                Text("현재 휴식을 마치거나 일시정지를 해제한 뒤 모드를 전환하세요.").font(.caption)
            }
            Button("포모도로 시간 설정") {
                work = vm.config.pomodoroWorkMin; shortBreak = vm.config.pomodoroBreakMin
                longBreak = vm.config.pomodoroLongBreakMin; every = vm.config.pomodoroLongBreakEvery
                editing = true
            }
            .disabled(vm.isSessionRunning || !canSwitch)
            if let message = actions.message { Text(message).font(.caption).foregroundStyle(.red).textSelection(.enabled) }
        }
        .confirmationDialog("현재 작업 구간을 초기화하고 모드를 전환할까요? 오늘 누계는 유지됩니다.", isPresented: $confirmSwitch) {
            Button("전환") { execute(pendingCommands) }
        }
        .sheet(isPresented: $editing) {
            VStack(alignment: .leading, spacing: 12) {
                Text("포모도로 시간").font(.headline)
                Stepper("집중 \(work)분", value: $work, in: 1...180)
                Stepper("짧은 휴식 \(shortBreak)분", value: $shortBreak, in: 1...60)
                Stepper("긴 휴식 \(longBreak)분", value: $longBreak, in: 1...120)
                Stepper("\(every)회마다 긴 휴식", value: $every, in: 1...12)
                HStack {
                    Button("취소") { editing = false }
                    Button("저장") {
                        execute([["config", "set", "pomodoro_work_min=\(work)", "pomodoro_break_min=\(shortBreak)", "pomodoro_long_break_min=\(longBreak)", "pomodoro_long_break_every=\(every)"]])
                        editing = false
                    }.disabled(vm.isSessionRunning || !canSwitch)
                }
            }.padding(20).frame(width: 310)
        }
    }

    private func request(_ commands: [[String]]) {
        guard !commands.isEmpty else { return }
        if vm.state.isSessionActive {
            pendingCommands = commands; confirmSwitch = true
        } else { execute(commands) }
    }
    private func execute(_ commands: [[String]]) {
        Task { @MainActor in
            await actions.perform(commands: commands, client: CLICommandClient(executablePath: findHelper("break-reminder")))
            vm.refresh()
        }
    }
}
