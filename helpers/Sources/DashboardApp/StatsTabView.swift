import SwiftUI
import Charts
import HelperCore

enum StatsPeriod: String, CaseIterable, Identifiable {
    case week = "주간"
    case month = "월간"
    case all = "전체"

    var id: String { rawValue }

    var days: Int {
        switch self {
        case .week: return 7
        case .month: return 30
        case .all: return 365
        }
    }
}

struct StatsTabView: View {
    @ObservedObject var vm: DashboardViewModel
    @State private var period: StatsPeriod = .week

    private var filteredHistory: [HistoryEntry] {
        let cutoff = Calendar.current.date(byAdding: .day, value: -period.days, to: Date()) ?? Date()
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let cutoffStr = formatter.string(from: cutoff)
        return vm.history.filter { $0.date >= cutoffStr }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                periodSelector
                workBreakChart
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 12)
        }
    }

    private var periodSelector: some View {
        Picker("Period", selection: $period) {
            ForEach(StatsPeriod.allCases) { p in
                Text(p.rawValue).tag(p)
            }
        }
        .pickerStyle(.segmented)
    }

    private var workBreakChart: some View {
        let workColor = Color(red: 0.3, green: 0.8, blue: 0.5)
        let breakColor = Color(red: 0.4, green: 0.7, blue: 1.0)

        return VStack(alignment: .leading, spacing: 8) {
            Text("작업 / 휴식 시간")
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(Color(white: 0.9))

            Chart {
                ForEach(filteredHistory, id: \.date) { entry in
                    BarMark(
                        x: .value("날짜", shortDate(entry.date)),
                        y: .value("분", entry.workMin)
                    )
                    .foregroundStyle(workColor)

                    BarMark(
                        x: .value("날짜", shortDate(entry.date)),
                        y: .value("분", entry.breakMin)
                    )
                    .foregroundStyle(breakColor)
                }
            }
            .frame(height: 140)

            HStack(spacing: 16) {
                HStack(spacing: 4) {
                    RoundedRectangle(cornerRadius: 2)
                        .fill(workColor)
                        .frame(width: 8, height: 8)
                    Text("작업").font(.system(size: 10)).foregroundColor(.gray)
                }
                HStack(spacing: 4) {
                    RoundedRectangle(cornerRadius: 2)
                        .fill(breakColor)
                        .frame(width: 8, height: 8)
                    Text("휴식").font(.system(size: 10)).foregroundColor(.gray)
                }
            }
        }
    }

    private func shortDate(_ iso: String) -> String {
        let parts = iso.split(separator: "-")
        guard parts.count == 3 else { return iso }
        return "\(Int(parts[1]) ?? 0)/\(Int(parts[2]) ?? 0)"
    }
}
