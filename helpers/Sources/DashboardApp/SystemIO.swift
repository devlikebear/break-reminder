import Foundation
import HelperCore

func loadStateFromDisk() -> AppState {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppState() }
    return parseState(from: content)
}

func loadConfigFromDisk() -> AppConfig {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".config/break-reminder/config.yaml")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return AppConfig() }
    return parseConfig(from: content)
}

func writeStateToDisk(_ s: AppState) {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-state")
    try? serializeState(s).data(using: .utf8)?.write(to: path, options: .atomic)
}

func queryLaunchdStatus() -> String {
    let task = Process()
    task.launchPath = "/bin/launchctl"
    task.arguments = ["list", "com.devlikebear.break-reminder"]
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = pipe
    do {
        try task.run()
        task.waitUntilExit()
        return task.terminationStatus == 0 ? "Running (launchd)" : "Not loaded"
    } catch {
        return "Unknown"
    }
}

func getIdleSecondsFromSystem() -> Int {
    let task = Process()
    task.launchPath = "/usr/sbin/ioreg"
    task.arguments = ["-c", "IOHIDSystem", "-d", "4"]
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = FileHandle.nullDevice
    do {
        try task.run()
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        task.waitUntilExit()
        guard let output = String(data: data, encoding: .utf8) else { return 0 }
        for line in output.components(separatedBy: "\n") {
            if line.contains("HIDIdleTime") {
                let parts = line.components(separatedBy: "=")
                if let last = parts.last {
                    let cleaned = last.trimmingCharacters(in: .whitespacesAndNewlines)
                    if let ns = Int64(cleaned) {
                        return Int(ns / 1_000_000_000)
                    }
                }
            }
        }
    } catch {}
    return 0
}

func findHelper(_ name: String) -> String? {
    return findHelper(name, checked: nil)
}

func findHelper(_ name: String, checked: UnsafeMutablePointer<[String]>?) -> String? {
    var candidates: [String] = []
    if let exe = Bundle.main.executablePath {
        candidates.append(
            URL(fileURLWithPath: exe)
                .deletingLastPathComponent()
                .appendingPathComponent(name)
                .path
        )
    }
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    candidates.append("\(home)/.local/bin/\(name)")
    if let checked = checked {
        checked.pointee = candidates
    }
    for candidate in candidates {
        if FileManager.default.isExecutableFile(atPath: candidate) {
            return candidate
        }
    }
    return nil
}

/// Append a timestamped line to ~/.break-reminder.log so GUI-side events show
/// alongside CLI logs. Best-effort — failures are silently dropped because we
/// don't want logging itself to crash the app.
func dashLog(_ message: String) {
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    let path = "\(home)/.break-reminder.log"
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd HH:mm:ss"
    let line = "[\(formatter.string(from: Date()))] [dashboard] \(message)\n"
    guard let data = line.data(using: .utf8) else { return }
    if let handle = FileHandle(forWritingAtPath: path) {
        defer { try? handle.close() }
        _ = try? handle.seekToEnd()
        try? handle.write(contentsOf: data)
    } else {
        try? data.write(to: URL(fileURLWithPath: path))
    }
}
