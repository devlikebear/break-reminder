import Foundation
import HelperCore

/// Locates bundled executables. Inside the .app they live in Contents/MacOS;
/// during development they can sit next to the binary or be named explicitly.
enum BundledTool {
    static func url(_ name: String, envOverride: String) -> URL? {
        let env = ProcessInfo.processInfo.environment
        if let path = env[envOverride], FileManager.default.isExecutableFile(atPath: path) {
            return URL(fileURLWithPath: path)
        }
        if let url = Bundle.main.url(forAuxiliaryExecutable: name), FileManager.default.isExecutableFile(atPath: url.path) {
            return url
        }
        if let exe = Bundle.main.executableURL {
            let sibling = exe.deletingLastPathComponent().appendingPathComponent(name)
            if FileManager.default.isExecutableFile(atPath: sibling.path) { return sibling }
        }
        return nil
    }

    /// The Go engine: `Contents/MacOS/engine`, or BREAK_REMINDER_ENGINE in development.
    static var engine: URL? { url("engine", envOverride: "BREAK_REMINDER_ENGINE") }
    static var breakScreen: URL? { url("break-screen", envOverride: "BREAK_REMINDER_BREAK_SCREEN") }
}

/// Environment for the engine: the UI language, and a data directory inside
/// the container when sandboxed (unless one was given explicitly).
func engineEnvironment() -> [String: String] {
    var env = ProcessInfo.processInfo.environment
    env["BREAK_REMINDER_LANGUAGE"] = AppLanguage.current.rawValue
    if env["BREAK_REMINDER_HOME"] == nil, env["APP_SANDBOX_CONTAINER_ID"] != nil,
       let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first {
        let home = support.appendingPathComponent("BreakReminder", isDirectory: true)
        try? FileManager.default.createDirectory(at: home, withIntermediateDirectories: true)
        env["BREAK_REMINDER_HOME"] = home.path
    }
    return env
}

/// Owns the engine connection and the latest status for the UI.
@MainActor
final class AppModel: ObservableObject {
    @Published private(set) var status: EngineStatus?
    @Published private(set) var engineRunning = false
    @Published private(set) var busy = false
    @Published var lastError: String?
    @Published private(set) var now = Date()

    let client: EngineClient?
    var onEvent: ((EngineEvent) -> Void)?
    /// Called each time an engine process comes up, so it gets idle time at once.
    var onEngineStarted: (() -> Void)?
    private var clockTimer: Timer?

    init() {
        if let engine = BundledTool.engine {
            client = EngineClient(configuration: .init(
                executableURL: engine,
                arguments: ["engine", "--stdio"],
                environment: engineEnvironment()
            ))
        } else {
            client = nil
            lastError = L10n.text("The timer engine is not running.")
        }
        client?.onStatus = { [weak self] status in self?.status = status }
        client?.onEvent = { [weak self] event in self?.onEvent?(event) }
        client?.onRunningChange = { [weak self] running in
            guard let self else { return }
            self.engineRunning = running
            if running {
                self.onEngineStarted?()
                Task { await self.refresh() }
            }
        }
        clockTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.now = Date() }
        }
    }

    func start() { client?.start() }
    func stop() { client?.stop() }

    var nowUnix: Int64 { Int64(now.timeIntervalSince1970) }

    var presentation: PopoverPresentation? {
        status.map { popoverPresentation(status: $0, now: nowUnix) }
    }

    func refresh() async {
        guard let client else { return }
        if let status = try? await client.call("status.get", as: EngineStatus.self) {
            self.status = status
        }
    }

    func reportIdle(_ seconds: Int) {
        guard let client, engineRunning else { return }
        Task { _ = try? await client.call("idle.report", params: ["seconds": seconds]) }
    }

    /// Runs one RPC; methods that return a status update the UI right away.
    func perform(_ method: String, _ params: [String: Any] = [:]) {
        guard let client else { return }
        busy = true
        Task {
            defer { busy = false }
            do {
                let data = try await client.call(method, params: params)
                if let status = try? JSONDecoder().decode(EngineStatus.self, from: data) { self.status = status }
                lastError = nil
            } catch {
                lastError = error.localizedDescription
            }
        }
    }

    func call(_ method: String, _ params: [String: Any] = [:]) async throws {
        guard let client else { throw EngineClientError.notRunning }
        _ = try await client.call(method, params: params)
    }
}
