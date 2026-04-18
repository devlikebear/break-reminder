import SwiftUI
import AppKit
import HelperCore

@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .frame(width: 360, height: 600)
                .background(Color(red: 0.1, green: 0.1, blue: 0.12))
                .onAppear {
                    vm.start()
                    configureWindow()
                }
                .onDisappear { vm.stop() }
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
    @FocusState private var isFocused: Bool
    @Environment(\.controlActiveState) private var controlActiveState

    private var isWindowActive: Bool {
        controlActiveState == .key || controlActiveState == .active
    }

    private var accentColor: Color {
        if vm.isPaused { return .yellow }
        return vm.isWork ? Color(red: 0.3, green: 0.8, blue: 0.5) : Color(red: 0.4, green: 0.7, blue: 1.0)
    }

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TabBarView(selectedTab: $vm.selectedTab, accentColor: accentColor)

            Group {
                switch vm.selectedTab {
                case .timer:
                    TimerTabView(vm: vm)
                case .stats:
                    StatsTabView(vm: vm)
                case .insights:
                    InsightsTabView(vm: vm)
                }
            }
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
        }
        .onChange(of: isWindowActive) { _, newValue in
            if newValue { isFocused = true }
        }
        .contentShape(Rectangle())
        .onTapGesture {
            NSApp.activate(ignoringOtherApps: true)
            if let window = NSApp.windows.first(where: { $0.title == "Break Reminder" }) {
                window.makeKeyAndOrderFront(nil)
            }
            isFocused = true
        }
    }

    private func installKeyMonitor() {
        NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            // Only handle plain keys (no modifiers like Cmd/Ctrl/Opt)
            let relevantFlags: NSEvent.ModifierFlags = [.command, .control, .option]
            if !event.modifierFlags.intersection(relevantFlags).isEmpty {
                return event
            }

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
