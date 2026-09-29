import SwiftUI
import AppKit
import HelperCore

@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()
    @StateObject private var theme = ThemeManager()
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .environmentObject(theme)
                .frame(width: 360, height: 600)
                .background(theme.background)
                .onAppear {
                    vm.start()
                    theme.mode = ThemeMode(raw: vm.config.theme)
                    configureWindow()
                }
                .onDisappear { vm.stop() }
                .onChange(of: vm.config.theme) { _, newValue in
                    theme.mode = ThemeMode(raw: newValue)
                }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
        .defaultPosition(.topTrailing)
    }

    private func configureWindow() {
        DispatchQueue.main.async {
            guard let window = NSApp.windows.first(where: { $0.title == "Break Reminder" }) else { return }
            window.level = .floating
            window.isMovableByWindowBackground = true
            window.titlebarAppearsTransparent = true
            window.titleVisibility = .hidden
            NSApp.activate(ignoringOtherApps: true)
            window.makeKeyAndOrderFront(nil)
        }
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
    }
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }
}

struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel
    @EnvironmentObject var theme: ThemeManager
    @FocusState private var isFocused: Bool
    @Environment(\.controlActiveState) private var controlActiveState
    @Environment(\.colorScheme) private var systemColorScheme
    @State private var keyMonitor: Any?
    @State private var confettiParticles: [ConfettiParticle] = []

    private var isWindowActive: Bool {
        controlActiveState == .key || controlActiveState == .active
    }

    private var accentColor: Color {
        if vm.isPaused { return theme.warning }
        return vm.isWork ? theme.accent : theme.accentBreak
    }

    var body: some View {
        ZStack {
            VStack(spacing: 0) {
                if vm.selectedTab == .timer && vm.selectedTimeTool == "타이머" {
                    HStack {
                        VStack(alignment: .leading, spacing: 3) {
                            Text("업무·휴식").font(.caption).foregroundStyle(.secondary)
                            Text(vm.statusText).font(.caption.weight(.semibold))
                        }
                        Spacer()
                        Button("집중 보기") { vm.selectedTimeTool = "집중" }
                    }.padding(16)
                } else {
                    StatusHeaderView(vm: vm)
                }
                Divider().background(theme.divider)
                TabBarView(selectedTab: $vm.selectedTab, accentColor: accentColor)

                Group {
                    switch vm.selectedTab {
                    case .timer:
                        TimerTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    case .stats:
                        StatsTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    case .insights:
                        InsightsTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    case .settings:
                        SettingsTabView(vm: vm)
                            .transition(.asymmetric(insertion: .move(edge: .trailing), removal: .move(edge: .leading)))
                    }
                }
                .animation(.easeInOut(duration: 0.25), value: vm.selectedTab)
            }

            ConfettiView(
                particles: confettiParticles,
                isActive: vm.showConfetti
            )
        }
        .opacity(isWindowActive ? 1.0 : 0.55)
        .animation(.easeInOut(duration: 0.2), value: isWindowActive)
        .focusable()
        .focused($isFocused)
        .focusEffectDisabled()
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
                isFocused = true
            }
            installKeyMonitor()
            theme.systemIsDark = (systemColorScheme == .dark)
            if confettiParticles.isEmpty {
                confettiParticles = ConfettiView.generate(
                    count: 50,
                    colors: [theme.accent, theme.accentBreak, theme.warning, .pink, .purple]
                )
            }
        }
        .onDisappear {
            if let keyMonitor { NSEvent.removeMonitor(keyMonitor) }
            keyMonitor = nil
        }
        .onChange(of: isWindowActive) { _, newValue in
            if newValue && !(NSApp.keyWindow?.firstResponder is NSTextView) { isFocused = true }
        }
        .onChange(of: systemColorScheme) { _, newValue in
            theme.systemIsDark = (newValue == .dark)
        }
        .contentShape(Rectangle())
        .onTapGesture {
            NSApp.activate(ignoringOtherApps: true)
            if let window = NSApp.windows.first(where: { $0.title == "Break Reminder" }) {
                window.makeKeyAndOrderFront(nil)
            }
            if !(NSApp.keyWindow?.firstResponder is NSTextView) { isFocused = true }
        }
    }

    private func installKeyMonitor() {
        guard keyMonitor == nil else { return }
        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            // Only handle plain keys (no modifiers like Cmd/Ctrl/Opt)
            let relevantFlags: NSEvent.ModifierFlags = [.command, .control, .option]
            let isEditing = event.window?.firstResponder is NSTextView
            if !shouldHandlePlainShortcut(isEditing: isEditing, hasModifiers: !event.modifierFlags.intersection(relevantFlags).isEmpty) {
                return event
            }

            if (event.keyCode == 15 || event.keyCode == 11) && (vm.selectedTab != .timer || vm.selectedTimeTool != "집중") { return event }

            switch event.keyCode {
            case 12:  // Q (physical key)
                NSApp.terminate(nil)
                return nil
            case 15:  // R
                vm.resetTimer()
                return nil
            case 11:  // B
                vm.forceBreak()
                return nil
            default:
                return event
            }
        }
    }
}
