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
                Divider().background(Color(white: 0.2))
                heatmapView
                Divider().background(Color(white: 0.2))
                summaryCards
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

    private var heatmapView: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("시간대별 집중도")
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(Color(white: 0.9))

            heatmapGrid
            heatmapLegend
        }
    }

    private var heatmapGrid: some View {
        let hours = Array(9...18)
        let entries = Array(filteredHistory.suffix(7))

        return VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 2) {
                Text("").frame(width: 28)
                ForEach(hours, id: \.self) { hour in
                    Text("\(hour)")
                        .font(.system(size: 9))
                        .foregroundColor(.gray)
                        .frame(maxWidth: .infinity)
                }
            }

            ForEach(entries, id: \.date) { entry in
                HStack(spacing: 2) {
                    Text(dayLabel(entry.date))
                        .font(.system(size: 9))
                        .foregroundColor(.gray)
                        .frame(width: 28, alignment: .leading)

                    ForEach(hours, id: \.self) { hour in
                        RoundedRectangle(cornerRadius: 2)
                            .fill(heatColor(for: entry.hourlyWork[hour]))
                            .frame(height: 14)
                    }
                }
            }
        }
    }

    private var heatmapLegend: some View {
        HStack(spacing: 4) {
            Text("낮음").font(.system(size: 9)).foregroundColor(.gray)
            ForEach([0, 15, 35, 55], id: \.self) { v in
                RoundedRectangle(cornerRadius: 2)
                    .fill(heatColor(for: v))
                    .frame(width: 12, height: 8)
            }
            Text("높음").font(.system(size: 9)).foregroundColor(.gray)
        }
    }

    private func heatColor(for minutes: Int) -> Color {
        switch minutes {
        case 0: return Color(red: 0.145, green: 0.145, blue: 0.157)
        case 1..<20: return Color(red: 0.102, green: 0.290, blue: 0.180)
        case 20..<45: return Color(red: 0.176, green: 0.478, blue: 0.290)
        default: return Color(red: 0.302, green: 0.800, blue: 0.502)
        }
    }

    private func dayLabel(_ iso: String) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let date = formatter.date(from: iso) else { return "" }

        let weekdayFormatter = DateFormatter()
        weekdayFormatter.locale = Locale(identifier: "ko_KR")
        weekdayFormatter.dateFormat = "E"
        return weekdayFormatter.string(from: date)
    }

    private var summaryCards: some View {
        let totalWork = filteredHistory.reduce(0) { $0 + $1.workMin }
        let totalBreak = filteredHistory.reduce(0) { $0 + $1.breakMin }
        let total = totalWork + totalBreak
        let ratio = total > 0 ? (totalWork * 100) / total : 0

        return HStack(spacing: 8) {
            summaryCard(label: "\(period.rawValue) 작업", value: formatMinutes(totalWork), color: Color(red: 0.3, green: 0.8, blue: 0.5))
            summaryCard(label: "\(period.rawValue) 휴식", value: formatMinutes(totalBreak), color: Color(red: 0.4, green: 0.7, blue: 1.0))
            summaryCard(label: "작업 비율", value: "\(ratio)%", color: Color(red: 1.0, green: 0.8, blue: 0.4))
        }
    }

    private func summaryCard(label: String, value: String, color: Color) -> some View {
        VStack(spacing: 4) {
            Text(value)
                .font(.system(size: 16, weight: .semibold))
                .foregroundColor(color)
            Text(label)
                .font(.system(size: 9))
                .foregroundColor(.gray)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 10)
        .background(Color(white: 0.15))
        .cornerRadius(8)
    }
}
