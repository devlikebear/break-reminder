import SwiftUI
import HelperCore

struct TimerTabView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            dailyStatsSection
            Divider().background(theme.divider)
            systemInfoSection
            Spacer()
            actionButtons
            shortcutHint
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }

    private var dailyStatsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Daily Statistics")
                .font(.system(size: 14, weight: .semibold))
                .foregroundColor(theme.textPrimary)

            let totals = vm.dailyTotals
            let workMin = totals.workSeconds / 60
            let breakMin = totals.breakSeconds / 60
            let totalMin = workMin + breakMin

            HStack {
                Text("Work: \(formatMinutes(workMin))")
                    .font(.system(size: 13))
                    .foregroundColor(theme.textPrimary)
                Spacer()
                Text("Break: \(formatMinutes(breakMin))")
                    .font(.system(size: 13))
                    .foregroundColor(theme.accentBreak)
            }

            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(theme.divider)
                        .frame(height: 6)
                    if totalMin > 0 {
                        RoundedRectangle(cornerRadius: 3)
                            .fill(theme.accent)
                            .frame(width: geo.size.width * CGFloat(workMin) / CGFloat(totalMin), height: 6)
                    }
                }
            }
            .frame(height: 6)

            if totalMin > 0 {
                HStack {
                    Spacer()
                    Text("\(workMin * 100 / totalMin)%")
                        .font(.system(size: 11))
                        .foregroundColor(theme.textSecondary)
                }
            }
        }
    }

    private var systemInfoSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("System: \(vm.launchdStatusText)")
                .font(.system(size: 12))
                .foregroundColor(theme.textSecondary)
            Text("Idle: \(vm.idleSeconds)s / Threshold: \(vm.config.idleThresholdSec)s")
                .font(.system(size: 12))
                .foregroundColor(theme.textSecondary)
        }
    }

    private var actionButtons: some View {
        HStack(spacing: 12) {
            Button("Reset") { vm.resetTimer() }
                .buttonStyle(DashboardButtonStyle())
            Button("Force Break") { vm.forceBreak() }
                .buttonStyle(DashboardButtonStyle())
        }
    }

    private var shortcutHint: some View {
        HStack {
            Spacer()
            Text("q: quit   r: reset   b: break")
                .font(.system(size: 10))
                .foregroundColor(theme.textSecondary)
            Spacer()
        }
    }
}

struct DashboardButtonStyle: ButtonStyle {
    let surfaceColor: Color
    let textColor: Color

    init(surfaceColor: Color = Color(white: 0.22), textColor: Color = Color(white: 0.9)) {
        self.surfaceColor = surfaceColor
        self.textColor = textColor
    }

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .medium))
            .foregroundColor(textColor)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(surfaceColor.opacity(configuration.isPressed ? 1.3 : 1.0))
            )
    }
}
