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
            Text(L10n.text("Activity-based · a long absence resets the focus interval."))
                .font(.caption).foregroundStyle(.secondary)
            HStack {
                Button(vm.config.pomodoroEnabled && vm.state.isSessionActive ? L10n.text("Pomodoro running") : L10n.text("Start Pomodoro")) {
                    request(pomodoroStartCommands(state: vm.state, config: vm.config))
                }
                .disabled(!canSwitch || (vm.config.pomodoroEnabled && vm.state.isSessionActive))
                if vm.config.pomodoroEnabled {
                    Button(L10n.text("Classic cycle")) { request([["pomodoro", "off"]]) }.disabled(!canSwitch)
                }
            }
            if vm.state.paused || vm.state.mode == "break" {
                Text(L10n.text("Finish your break or resume before switching modes.")).font(.caption)
            }
            Button(L10n.text("Pomodoro durations")) {
                work = vm.config.pomodoroWorkMin; shortBreak = vm.config.pomodoroBreakMin
                longBreak = vm.config.pomodoroLongBreakMin; every = vm.config.pomodoroLongBreakEvery
                editing = true
            }
            .disabled(vm.isSessionRunning || !canSwitch)
            if let message = actions.message { Text(message).font(.caption).foregroundStyle(.red).textSelection(.enabled) }
        }
        .confirmationDialog(L10n.text("Reset the current work interval and switch modes? Today's totals will be kept."), isPresented: $confirmSwitch) {
            Button(L10n.text("Switch")) { execute(pendingCommands) }
        }
        .sheet(isPresented: $editing) {
            VStack(alignment: .leading, spacing: 12) {
                Text(L10n.text("Pomodoro durations")).font(.headline)
                Stepper(L10n.text("Focus: {0} min", work), value: $work, in: 1...180)
                Stepper(L10n.text("Short break: {0} min", shortBreak), value: $shortBreak, in: 1...60)
                Stepper(L10n.text("Long break: {0} min", longBreak), value: $longBreak, in: 1...120)
                Stepper(L10n.text("Long break every {0} cycles", every), value: $every, in: 1...12)
                HStack {
                    Button(L10n.text("Cancel")) { editing = false }
                    Button(L10n.text("Save")) {
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
