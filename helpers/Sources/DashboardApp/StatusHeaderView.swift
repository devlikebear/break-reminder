import SwiftUI
import HelperCore

struct StatusHeaderView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager

    private var statusColor: Color {
        if vm.isPaused { return theme.warning }
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

            mascotRow
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
        .animation(.easeInOut(duration: 0.5), value: vm.isWork)
        .animation(.easeInOut(duration: 0.3), value: vm.isPaused)
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
