import SwiftUI
import HelperCore

struct StatusHeaderView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager

    private var statusColor: Color {
        if vm.isPaused { return theme.warning }
        if !vm.isSessionRunning { return theme.textSecondary }
        return vm.isWork ? theme.accent : theme.accentBreak
    }

    private var ringSize: CGFloat { 140 }

    var body: some View {
        VStack(spacing: 12) {
            HStack {
                Circle()
                    .fill(statusColor)
                    .frame(width: 10, height: 10)
                Text(vm.statusText)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundColor(statusColor)
                Spacer()
                Text(vm.modeDetail)
                    .font(.system(size: 12))
                    .foregroundColor(theme.textSecondary)
            }

            ZStack {
                CircularProgressRing(
                    progress: vm.sessionProgress.progress,
                    fillColor: statusColor,
                    trackColor: theme.divider,
                    lineWidth: 10
                )
                .frame(width: ringSize, height: ringSize)

                VStack(spacing: 2) {
                    Text(vm.sessionProgress.remainingFormatted)
                        .font(.system(size: 32, weight: .ultraLight).monospacedDigit())
                        .foregroundColor(theme.textPrimary)
                    Text(vm.sessionSubtitle)
                        .font(.system(size: 11))
                        .foregroundColor(theme.textSecondary)
                }
            }

            pauseControlRow

            mascotRow
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
        .animation(.easeInOut(duration: 0.5), value: vm.isWork)
        .animation(.easeInOut(duration: 0.3), value: vm.isPaused)
    }

    private var pauseControlRow: some View {
        HStack(spacing: 8) {
            if vm.isPaused {
                Circle()
                    .fill(vm.pauseModeAccent)
                    .frame(width: 8, height: 8)
                Text(vm.pauseModeLabel.isEmpty ? "Paused" : L10n.text("Paused · {0}", vm.pauseModeLabel))
                    .font(.system(size: 12, weight: .medium))
                    .foregroundColor(vm.pauseModeAccent)
                if let remaining = vm.pauseRemainingText {
                    Text("· \(remaining)")
                        .font(.system(size: 12).monospacedDigit())
                        .foregroundColor(theme.textSecondary)
                }
                Spacer()
                Button {
                    vm.resume()
                } label: {
                    Label(L10n.text("Resume"), systemImage: "play.fill")
                        .font(.system(size: 12, weight: .medium))
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
            } else {
                if !vm.isSessionRunning {
                    Circle()
                        .fill(theme.textSecondary)
                        .frame(width: 8, height: 8)
                    Text(L10n.text("Work session off"))
                        .font(.system(size: 12, weight: .medium))
                        .foregroundColor(theme.textSecondary)
                }
                Spacer()
                Button {
                    vm.toggleSession()
                } label: {
                    Label(vm.isSessionRunning ? L10n.text("Stop") : "Start",
                          systemImage: vm.isSessionRunning ? "stop.fill" : "play.fill")
                        .font(.system(size: 12, weight: .medium))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                Menu {
                    pauseModeMenu(mode: "meeting", title: L10n.text("Meeting"), icon: "person.2.fill")
                    pauseModeMenu(mode: "focus", title: L10n.text("Focus"), icon: "bolt.fill")
                    pauseModeMenu(mode: "afk", title: L10n.text("Away"), icon: "figure.walk")
                } label: {
                    Label(L10n.text("Pause"), systemImage: "pause.fill")
                        .font(.system(size: 12, weight: .medium))
                }
                .menuStyle(.borderlessButton)
                .controlSize(.small)
                .fixedSize()
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(theme.surface)
        )
    }

    @ViewBuilder
    private func pauseModeMenu(mode: String, title: String, icon: String) -> some View {
        Menu {
            Button(L10n.text("15 min")) { vm.pause(mode: mode, durationMinutes: 15) }
            Button(L10n.text("30 min")) { vm.pause(mode: mode, durationMinutes: 30) }
            Button(L10n.text("1 hour")) { vm.pause(mode: mode, durationMinutes: 60) }
            Button(L10n.text("2 hours")) { vm.pause(mode: mode, durationMinutes: 120) }
            Divider()
            Button(L10n.text("Indefinitely")) { vm.pause(mode: mode, durationMinutes: nil) }
        } label: {
            Label(title, systemImage: icon)
        }
    }

    private var mascotRow: some View {
        HStack(spacing: 8) {
            Text(vm.currentMascot.emoji)
                .font(.system(size: 22))
                .scaleEffect(vm.isPaused ? 0.9 : 1.0)
                .animation(.spring(response: 0.4, dampingFraction: 0.6), value: vm.currentMascot.emoji)
                .id(vm.currentMascot.emoji) // Trigger transition on emoji change

            Text(vm.currentMascot.message)
                .font(.system(size: 11))
                .foregroundColor(theme.textSecondary)
                .lineLimit(2)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(theme.surface)
        )
        .frame(maxWidth: .infinity)
    }
}
