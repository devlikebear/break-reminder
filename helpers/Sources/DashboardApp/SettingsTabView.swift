import SwiftUI
import HelperCore

final class SettingsFormState: ObservableObject {
    @Published var workDurationMin: Int
    @Published var breakDurationMin: Int
    @Published var idleThresholdSec: Int
    @Published var naturalBreakSec: Int
    @Published var workStartHour: Int
    @Published var workStartMinute: Int
    @Published var workEndHour: Int
    @Published var workEndMinute: Int
    @Published var workDays: Set<Int>
    @Published var breakScreenMode: String
    @Published var notificationsEnabled: Bool
    @Published var ttsEnabled: Bool
    @Published var breakActivitiesEnabled: Bool
    @Published var theme: String

    private(set) var original: AppConfig

    init(from cfg: AppConfig) {
        self.workDurationMin = cfg.workDurationMin
        self.breakDurationMin = cfg.breakDurationMin
        self.idleThresholdSec = cfg.idleThresholdSec
        self.naturalBreakSec = cfg.naturalBreakSec
        self.workStartHour = cfg.workStartHour
        self.workStartMinute = cfg.workStartMinute
        self.workEndHour = cfg.workEndHour
        self.workEndMinute = cfg.workEndMinute
        self.workDays = Set(cfg.workDays)
        self.breakScreenMode = cfg.breakScreenMode
        self.notificationsEnabled = cfg.notificationsEnabled
        self.ttsEnabled = cfg.ttsEnabled
        self.breakActivitiesEnabled = cfg.breakActivitiesEnabled
        self.theme = cfg.theme
        self.original = cfg
    }

    func reset(to cfg: AppConfig) {
        workDurationMin = cfg.workDurationMin
        breakDurationMin = cfg.breakDurationMin
        idleThresholdSec = cfg.idleThresholdSec
        naturalBreakSec = cfg.naturalBreakSec
        workStartHour = cfg.workStartHour
        workStartMinute = cfg.workStartMinute
        workEndHour = cfg.workEndHour
        workEndMinute = cfg.workEndMinute
        workDays = Set(cfg.workDays)
        breakScreenMode = cfg.breakScreenMode
        notificationsEnabled = cfg.notificationsEnabled
        ttsEnabled = cfg.ttsEnabled
        breakActivitiesEnabled = cfg.breakActivitiesEnabled
        theme = cfg.theme
        original = cfg
    }

    func revert() {
        reset(to: original)
    }

    var isDirty: Bool {
        return workDurationMin != original.workDurationMin
            || breakDurationMin != original.breakDurationMin
            || idleThresholdSec != original.idleThresholdSec
            || naturalBreakSec != original.naturalBreakSec
            || workStartHour != original.workStartHour
            || workStartMinute != original.workStartMinute
            || workEndHour != original.workEndHour
            || workEndMinute != original.workEndMinute
            || workDays != Set(original.workDays)
            || breakScreenMode != original.breakScreenMode
            || notificationsEnabled != original.notificationsEnabled
            || ttsEnabled != original.ttsEnabled
            || breakActivitiesEnabled != original.breakActivitiesEnabled
            || theme != original.theme
    }

    /// Encodes the full form state into CLI key=value pairs suitable for
    /// `break-reminder config set`. All keys are sent every time; the CLI
    /// validation is idempotent on unchanged values.
    func toChanges() -> [(String, String)] {
        let sortedDays = workDays.sorted()
        let workDaysArg = "[" + sortedDays.map(String.init).joined(separator: ",") + "]"
        return [
            ("work_duration_min", String(workDurationMin)),
            ("break_duration_min", String(breakDurationMin)),
            ("idle_threshold_sec", String(idleThresholdSec)),
            ("natural_break_sec", String(naturalBreakSec)),
            ("work_start_hour", String(workStartHour)),
            ("work_start_minute", String(workStartMinute)),
            ("work_end_hour", String(workEndHour)),
            ("work_end_minute", String(workEndMinute)),
            ("work_days", workDaysArg),
            ("break_screen_mode", breakScreenMode),
            ("notifications_enabled", notificationsEnabled ? "true" : "false"),
            ("tts_enabled", ttsEnabled ? "true" : "false"),
            ("break_activities_enabled", breakActivitiesEnabled ? "true" : "false"),
            ("theme", theme),
        ]
    }
}

private let weekdayLabels: [(Int, String)] = [
    (1, "월"), (2, "화"), (3, "수"), (4, "목"), (5, "금"), (6, "토"), (7, "일")
]

struct SettingsTabView: View {
    @ObservedObject var vm: DashboardViewModel
    @StateObject private var form: SettingsFormState
    @State private var showError = false
    @State private var errorMessage = ""
    @State private var showSavedToast = false
    @EnvironmentObject var theme: ThemeManager

    init(vm: DashboardViewModel) {
        self.vm = vm
        _form = StateObject(wrappedValue: SettingsFormState(from: vm.config))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                timerSection
                scheduleSection
                behaviorSection
                themeSection
                buttonRow
                if showSavedToast {
                    Text("저장됨")
                        .font(.system(size: 11))
                        .foregroundColor(theme.accentBreak)
                }
            }
            .padding(16)
        }
        .background(theme.background)
        .alert("저장 실패", isPresented: $showError) {
            Button("확인", role: .cancel) {}
        } message: {
            Text(errorMessage)
        }
        .onAppear {
            form.reset(to: vm.config)
        }
    }

    private var timerSection: some View {
        sectionCard(title: "타이머") {
            stepperRow("작업 시간 (work_duration_min)", value: $form.workDurationMin, range: 5...180, suffix: "분")
            stepperRow("휴식 시간 (break_duration_min)", value: $form.breakDurationMin, range: 1...60, suffix: "분")
            stepperRow("Idle 임계값 (idle_threshold_sec)", value: $form.idleThresholdSec, range: 30...600, step: 10, suffix: "초")
            stepperRow("Natural break (natural_break_sec)", value: $form.naturalBreakSec, range: 60...1800, step: 30, suffix: "초")
        }
    }

    private var scheduleSection: some View {
        sectionCard(title: "근무 시간") {
            HStack {
                Text("시작 (work_start_*)")
                    .frame(maxWidth: .infinity, alignment: .leading)
                Stepper(value: $form.workStartHour, in: 0...23) {
                    Text(String(format: "%02d:%02d", form.workStartHour, form.workStartMinute))
                        .monospacedDigit()
                }
                Stepper(value: $form.workStartMinute, in: 0...59, step: 5) {
                    EmptyView()
                }
                .labelsHidden()
            }
            HStack {
                Text("종료 (work_end_*)")
                    .frame(maxWidth: .infinity, alignment: .leading)
                Stepper(value: $form.workEndHour, in: 0...23) {
                    Text(String(format: "%02d:%02d", form.workEndHour, form.workEndMinute))
                        .monospacedDigit()
                }
                Stepper(value: $form.workEndMinute, in: 0...59, step: 5) {
                    EmptyView()
                }
                .labelsHidden()
            }
            Text("근무 요일 (work_days)")
                .frame(maxWidth: .infinity, alignment: .leading)
            HStack(spacing: 4) {
                ForEach(weekdayLabels, id: \.0) { day, label in
                    Toggle(isOn: bindingForDay(day)) {
                        Text(label)
                            .font(.system(size: 11))
                    }
                    .toggleStyle(.button)
                }
            }
        }
    }

    private var behaviorSection: some View {
        sectionCard(title: "동작") {
            Picker("Break screen mode (break_screen_mode)", selection: $form.breakScreenMode) {
                Text("ask").tag("ask")
                Text("block").tag("block")
                Text("notify").tag("notify")
            }
            .pickerStyle(.segmented)
            Toggle("알림 (notifications_enabled)", isOn: $form.notificationsEnabled)
            Toggle("TTS (tts_enabled)", isOn: $form.ttsEnabled)
            Toggle("휴식 활동 (break_activities_enabled)", isOn: $form.breakActivitiesEnabled)
        }
    }

    private var themeSection: some View {
        sectionCard(title: "테마") {
            Picker("테마 (theme)", selection: $form.theme) {
                Text("auto").tag("auto")
                Text("dark").tag("dark")
                Text("light").tag("light")
            }
            .pickerStyle(.segmented)
        }
    }

    private var buttonRow: some View {
        HStack {
            Button("취소") { form.revert() }
                .disabled(!form.isDirty)
            Spacer()
            Button("기본값 복원") { form.reset(to: AppConfig()) }
            Button("저장") {
                let result = vm.saveSettings(form.toChanges())
                switch result {
                case .success:
                    form.reset(to: vm.config)
                    showSavedToast = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) {
                        showSavedToast = false
                    }
                case .failure(let err):
                    errorMessage = err.localizedDescription
                    showError = true
                }
            }
            .keyboardShortcut(.defaultAction)
            .disabled(!form.isDirty)
        }
    }

    // MARK: - Helpers

    private func bindingForDay(_ day: Int) -> Binding<Bool> {
        Binding(
            get: { form.workDays.contains(day) },
            set: { isOn in
                if isOn { form.workDays.insert(day) }
                else { form.workDays.remove(day) }
            }
        )
    }

    @ViewBuilder
    private func sectionCard<Content: View>(title: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title)
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(theme.textPrimary)
            VStack(alignment: .leading, spacing: 8) {
                content()
            }
            .padding(12)
            .background(
                RoundedRectangle(cornerRadius: 10)
                    .fill(theme.surface)
            )
        }
    }

    private func stepperRow(_ label: String, value: Binding<Int>, range: ClosedRange<Int>, step: Int = 1, suffix: String) -> some View {
        HStack {
            Text(label)
                .frame(maxWidth: .infinity, alignment: .leading)
            Stepper(value: value, in: range, step: step) {
                Text("\(value.wrappedValue) \(suffix)")
                    .monospacedDigit()
            }
            .fixedSize()
        }
    }
}
