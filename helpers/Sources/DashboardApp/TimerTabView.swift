import SwiftUI
import HelperCore

struct TimerTabView: View {
    @ObservedObject var vm: DashboardViewModel

    private let workColor = Color(red: 0.3, green: 0.8, blue: 0.5)
    private let breakColor = Color(red: 0.4, green: 0.7, blue: 1.0)

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            dailyStatsSection
            Divider().background(Color(white: 0.2))
            systemInfoSection
            Spacer()
            actionButtons
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }

    private var dailyStatsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Daily Statistics")
                .font(.system(size: 14, weight: .semibold))
                .foregroundColor(Color(white: 0.9))

            let totals = vm.dailyTotals
            let workMin = totals.workSeconds / 60
            let breakMin = totals.breakSeconds / 60
            let totalMin = workMin + breakMin

            HStack {
                Text("Work: \(formatMinutes(workMin))")
                    .font(.system(size: 13))
                    .foregroundColor(Color(white: 0.9))
                Spacer()
                Text("Break: \(formatMinutes(breakMin))")
                    .font(.system(size: 13))
                    .foregroundColor(breakColor)
            }

            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(Color(white: 0.2))
                        .frame(height: 6)
                    if totalMin > 0 {
                        RoundedRectangle(cornerRadius: 3)
                            .fill(workColor)
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
                        .foregroundColor(.gray)
                }
            }
        }
    }

    private var systemInfoSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("System: \(vm.launchdStatusText)")
                .font(.system(size: 12))
                .foregroundColor(.gray)
            Text("Idle: \(vm.idleSeconds)s / Threshold: \(vm.config.idleThresholdSec)s")
                .font(.system(size: 12))
                .foregroundColor(.gray)
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
}

struct DashboardButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .medium))
            .foregroundColor(Color(white: 0.9))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color(white: configuration.isPressed ? 0.28 : 0.22))
            )
    }
}
