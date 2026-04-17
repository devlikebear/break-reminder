import SwiftUI
import AppKit
import HelperCore

@main
struct DashboardAppEntry: App {
    @StateObject private var vm = DashboardViewModel()

    var body: some Scene {
        Window("Break Reminder", id: "dashboard") {
            DashboardContentView(vm: vm)
                .frame(width: 360, height: 600)
                .background(Color(red: 0.1, green: 0.1, blue: 0.12))
                .onAppear { vm.start() }
                .onDisappear { vm.stop() }
                .onKeyPress("q") { NSApp.terminate(nil); return .handled }
                .onKeyPress("r") { vm.resetTimer(); return .handled }
                .onKeyPress("b") { vm.forceBreak(); return .handled }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
        .defaultPosition(.topTrailing)
    }
}

struct DashboardContentView: View {
    @ObservedObject var vm: DashboardViewModel

    var body: some View {
        VStack(spacing: 0) {
            StatusHeaderView(vm: vm)
            Divider().background(Color(white: 0.2))
            TimerTabView(vm: vm)
        }
    }
}
