import SwiftUI

struct TabBarView: View {
    @Binding var selectedTab: DashboardTab
    let accentColor: Color
    @EnvironmentObject var theme: ThemeManager

    var body: some View {
        HStack(spacing: 0) {
            ForEach(DashboardTab.allCases) { tab in
                Button(action: { selectedTab = tab }) {
                    VStack(spacing: 6) {
                        Text(tab.rawValue)
                            .font(.system(size: 13, weight: selectedTab == tab ? .semibold : .regular))
                            .foregroundColor(selectedTab == tab ? accentColor : theme.textSecondary)
                        Rectangle()
                            .fill(selectedTab == tab ? accentColor : Color.clear)
                            .frame(height: 2)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 10)
                }
                .buttonStyle(.plain)
            }
        }
        .background(theme.background)
        .overlay(
            Rectangle()
                .fill(theme.divider)
                .frame(height: 1),
            alignment: .bottom
        )
    }
}
