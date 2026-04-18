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
                Text("Loading charts...")
                    .foregroundColor(.gray)
                    .font(.system(size: 12))
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
}
