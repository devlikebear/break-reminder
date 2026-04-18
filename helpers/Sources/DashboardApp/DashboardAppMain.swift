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
                    installKeyMonitor()
                }
                .onDisappear { vm.stop() }
                .onReceive(NotificationCenter.default.publisher(for: NSWindow.didBecomeKeyNotification)) { _ in
                    vm.isWindowActive = true
                }
                .onReceive(NotificationCenter.default.publisher(for: NSWindow.didResignKeyNotification)) { _ in
                    vm.isWindowActive = false
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

    private func installKeyMonitor() {
        NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            switch event.charactersIgnoringModifiers {
            case "q": NSApp.terminate(nil); return nil
            case "r": vm.resetTimer(); return nil
            case "b": vm.forceBreak(); return nil
            default: return event
            }
        }
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }
}

struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TimerTabView(vm: vm)
        }
        .opacity(vm.isWindowActive ? 1.0 : 0.55)
        .animation(.easeInOut(duration: 0.2), value: vm.isWindowActive)
        .contentShape(Rectangle())
        .onTapGesture {
            NSApp.activate(ignoringOtherApps: true)
            if let window = NSApp.windows.first(where: { $0.title == "Break Reminder" }) {
                window.makeKeyAndOrderFront(nil)
            }
        }
    }
}
