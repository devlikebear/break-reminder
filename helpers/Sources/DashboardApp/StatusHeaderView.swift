import SwiftUI
import HelperCore

struct StatusHeaderView: View {
    @ObservedObject var vm: DashboardViewModel

    private var statusColor: Color {
        if vm.isPaused { return .yellow }
        return vm.isWork ? Color(red: 0.3, green: 0.8, blue: 0.5) : Color(red: 0.4, green: 0.7, blue: 1.0)
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
                    .foregroundColor(.gray)
            }

            ZStack {
                CircularProgressRing(
                    progress: vm.sessionProgress.progress,
                    fillColor: statusColor,
                    lineWidth: 10
                )
                .frame(width: ringSize, height: ringSize)

                VStack(spacing: 2) {
                    Text(vm.sessionProgress.remainingFormatted)
                        .font(.system(size: 32, weight: .ultraLight).monospacedDigit())
                        .foregroundColor(Color(white: 0.9))
                    Text(vm.sessionSubtitle)
                        .font(.system(size: 11))
                        .foregroundColor(.gray)
                }
            }
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
    }
}
