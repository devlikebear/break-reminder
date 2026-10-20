import AppKit
import CoreGraphics
import EventKit
import Foundation
import HelperCore
import ServiceManagement

/// Phase 0 sandbox spike (docs/plans/renewal-spike-sandbox.md). Runs only when
/// BREAK_REMINDER_SPIKE=1 and writes spike-report.json into the data dir.
@MainActor
enum SpikeProbe {
    static var enabled: Bool { ProcessInfo.processInfo.environment["BREAK_REMINDER_SPIKE"] == "1" }

    static func run(model: AppModel) async {
        var report: [String: Any] = [:]
        let env = engineEnvironment()
        let home = URL(fileURLWithPath: env["BREAK_REMINDER_HOME"] ?? NSTemporaryDirectory())
        report["sandboxed"] = env["APP_SANDBOX_CONTAINER_ID"] != nil
        report["data_dir"] = home.path

        report["idle_seconds"] = IdleReporter.idleSeconds()

        let calendar = EKEventStore.authorizationStatus(for: .event)
        report["calendar_status_before"] = String(describing: calendar.rawValue)
        if ProcessInfo.processInfo.environment["BREAK_REMINDER_SPIKE_CALENDAR"] == "1" {
            let store = EKEventStore()
            do {
                let granted = try await store.requestFullAccessToEvents()
                report["calendar_request"] = granted ? "granted" : "denied"
                if granted {
                    let start = Date(), end = start.addingTimeInterval(86_400)
                    let events = store.events(matching: store.predicateForEvents(withStart: start, end: end, calendars: nil))
                    report["calendar_events_next_24h"] = events.count
                }
            } catch {
                report["calendar_request"] = "error: \(error.localizedDescription)"
            }
        }

        report["login_item_status"] = String(describing: SMAppService.mainApp.status.rawValue)

        // The sidecar is already running through the normal path.
        try? await Task.sleep(nanoseconds: 3_000_000_000)
        report["engine_running"] = model.engineRunning
        report["engine_status_received"] = model.status != nil
        report["engine_files"] = (try? FileManager.default.contentsOfDirectory(atPath: home.path))?.sorted() ?? []

        if let screen = BundledTool.breakScreen {
            let process = Process()
            process.executableURL = screen
            process.arguments = ["--duration", "3", "--skip-after", "120", "--work-min", "0", "--break-min", "0"]
            let started = Date()
            do {
                try process.run()
                process.waitUntilExit()
                report["break_screen"] = "exit \(process.terminationStatus) after \(Int(Date().timeIntervalSince(started)))s, screens=\(NSScreen.screens.count)"
            } catch {
                report["break_screen"] = "launch failed: \(error.localizedDescription)"
            }
        }

        if let probe = BundledTool.url("sandbox-probe", envOverride: "BREAK_REMINDER_PROBE") {
            let process = Process()
            process.executableURL = probe
            process.environment = env
            let out = Pipe()
            process.standardOutput = out
            process.standardError = out
            do {
                try process.run()
                let data = out.fileHandleForReading.readDataToEndOfFile()
                process.waitUntilExit()
                report["engine_probe"] = (try? JSONSerialization.jsonObject(with: data)) ?? String(decoding: data, as: UTF8.self)
            } catch {
                report["engine_probe"] = "launch failed: \(error.localizedDescription)"
            }
        }

        if let data = try? JSONSerialization.data(withJSONObject: report, options: [.prettyPrinted, .sortedKeys]) {
            try? data.write(to: home.appendingPathComponent("spike-report.json"))
        }
        NSApp.terminate(nil)
    }
}
