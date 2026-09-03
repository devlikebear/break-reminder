import SwiftUI
import AppKit
import HelperCore

struct InsightsTabView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                refreshStatusView
                if let report = vm.insights {
                    dailyReportCard(report)
                    Divider().background(theme.divider)
                    patternsSection(report)
                    Divider().background(theme.divider)
                    actionButtons(report)
                } else {
                    emptyState
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 12)
        }
        .scrollIndicators(.visible)
    }

    @ViewBuilder
    private var refreshStatusView: some View {
        switch vm.insightsRefreshStatus {
        case .idle:
            EmptyView()
        case .running:
            refreshStatusCard(color: theme.accent) {
                HStack(spacing: 8) {
                    ProgressView()
                        .controlSize(.small)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("AI 분석을 실행 중입니다")
                            .font(.system(size: 11, weight: .medium))
                        Text("최대 2분 정도 걸릴 수 있습니다.")
                            .font(.system(size: 10))
                            .foregroundColor(.secondary)
                    }
                    Spacer()
                }
            }
        case .succeeded:
            refreshStatusCard(color: theme.accent) {
                Label("AI 분석이 완료되었습니다", systemImage: "checkmark.circle.fill")
                    .font(.system(size: 11, weight: .medium))
            }
        case .failed(let message):
            refreshStatusCard(color: .red) {
                VStack(alignment: .leading, spacing: 8) {
                    Label("AI 분석 실패", systemImage: "exclamationmark.triangle.fill")
                        .font(.system(size: 11, weight: .semibold))
                    Text(message)
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(.primary)
                        .fixedSize(horizontal: false, vertical: true)
                        .textSelection(.enabled)
                    Text("오류 내용을 확인하고 설정 또는 CLI를 수정한 뒤 다시 시도하세요.")
                        .font(.system(size: 10))
                        .foregroundColor(.secondary)
                }
            }
        }
    }

    private func refreshStatusCard<Content: View>(color: Color, @ViewBuilder content: () -> Content) -> some View {
        content()
            .foregroundColor(color)
            .padding(10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(color.opacity(theme.isDark ? 0.16 : 0.08))
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(color.opacity(theme.isDark ? 0.45 : 0.3), lineWidth: 1)
            )
            .cornerRadius(8)
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "sparkles")
                .font(.system(size: 40))
                .foregroundColor(.gray)
            Text("아직 인사이트가 없습니다")
                .font(.system(size: 13))
                .foregroundColor(.primary)
            Text("AI CLI(claude 또는 codex)가 설치되어 있다면\n아래 버튼을 눌러 생성하세요.")
                .font(.system(size: 11))
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
            Button(action: { vm.refreshInsights() }) {
                HStack(spacing: 4) {
                    if vm.isRefreshingInsights {
                        ProgressView().controlSize(.small)
                    }
                    Text(vm.isRefreshingInsights ? "분석 중…" : "AI 분석 생성")
                }
            }
            .buttonStyle(DashboardButtonStyle())
            .frame(width: 140)
            .disabled(vm.isRefreshingInsights)
        }
        .frame(maxWidth: .infinity, minHeight: 200)
        .padding(.top, 40)
    }

    private func dailyReportCard(_ report: InsightsReport) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("✨ 오늘의 리포트")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(.primary)
                Spacer()
                Text(shortTime(report.generatedAt))
                    .font(.system(size: 10))
                    .foregroundColor(.secondary)
            }

            HStack(alignment: .top, spacing: 8) {
                Rectangle()
                    .fill(theme.accent)
                    .frame(width: 3)
                Text(report.dailyReport)
                    .font(.system(size: 12))
                    .foregroundColor(.primary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(theme.surface)
            .cornerRadius(10)
        }
    }

    private func patternsSection(_ report: InsightsReport) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("🔍 패턴 인사이트")
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(.primary)

            ForEach(Array(report.patterns.enumerated()), id: \.offset) { _, p in
                patternCard(p)
            }
        }
    }

    private func patternCard(_ pattern: InsightPattern) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Circle()
                    .fill(patternColor(pattern.type))
                    .frame(width: 6, height: 6)
                Text(pattern.title)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundColor(.primary)
            }
            Text(pattern.description)
                .font(.system(size: 11))
                .foregroundColor(.secondary)
                .fixedSize(horizontal: false, vertical: true)
            if !pattern.suggestion.isEmpty {
                Text("→ \(pattern.suggestion)")
                    .font(.system(size: 11))
                    .foregroundColor(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(theme.surface)
        .cornerRadius(8)
    }

    private func actionButtons(_ report: InsightsReport) -> some View {
        HStack(spacing: 12) {
            Button(action: { vm.refreshInsights() }) {
                HStack(spacing: 4) {
                    if vm.isRefreshingInsights {
                        ProgressView().controlSize(.small)
                    } else {
                        Text("🔄")
                    }
                    Text(vm.isRefreshingInsights ? "분석 중…" : "새로고침")
                }
            }
            .buttonStyle(DashboardButtonStyle())
            .disabled(vm.isRefreshingInsights)

            Button(action: { copyReport(report) }) {
                Text("📋 리포트 복사")
            }
            .buttonStyle(DashboardButtonStyle())
        }
    }

    private func patternColor(_ type: String) -> Color {
        switch type {
        case "warning": return Color(red: 1.0, green: 0.8, blue: 0.4)
        case "positive": return Color(red: 0.3, green: 0.8, blue: 0.5)
        default: return Color(red: 0.4, green: 0.7, blue: 1.0)
        }
    }

    private func shortTime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return iso }
        let display = DateFormatter()
        display.dateFormat = "HH:mm"
        return "\(display.string(from: date)) 생성"
    }

    private func copyReport(_ report: InsightsReport) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        pasteboard.setString(report.dailyReport, forType: .string)
    }
}
