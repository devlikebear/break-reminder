import SwiftUI

enum ThemeMode: String {
    case auto, dark, light

    init(raw: String) {
        self = ThemeMode(rawValue: raw) ?? .auto
    }
}

@MainActor
final class ThemeManager: ObservableObject {
    @Published var mode: ThemeMode = .auto
    @Published var systemIsDark: Bool = true

    var isDark: Bool {
        switch mode {
        case .dark: return true
        case .light: return false
        case .auto: return systemIsDark
        }
    }

    // MARK: - Color tokens

    var background: Color {
        isDark ? Color(red: 0.102, green: 0.102, blue: 0.118) : Color(red: 0.961, green: 0.961, blue: 0.969)
    }

    var surface: Color {
        isDark ? Color(red: 0.145, green: 0.145, blue: 0.157) : Color.white
    }

    var textPrimary: Color {
        isDark ? Color(white: 0.9) : Color(red: 0.102, green: 0.102, blue: 0.118)
    }

    var textSecondary: Color {
        isDark ? Color(white: 0.5) : Color(white: 0.4)
    }

    var accent: Color {
        isDark ? Color(red: 0.302, green: 0.800, blue: 0.502) : Color(red: 0.204, green: 0.659, blue: 0.325)
    }

    var accentBreak: Color {
        isDark ? Color(red: 0.400, green: 0.702, blue: 1.000) : Color(red: 0.259, green: 0.522, blue: 0.957)
    }

    var warning: Color {
        isDark ? Color(red: 1.0, green: 0.8, blue: 0.4) : Color(red: 0.976, green: 0.671, blue: 0.000)
    }

    var divider: Color {
        Color(white: isDark ? 0.2 : 0.85)
    }
}
