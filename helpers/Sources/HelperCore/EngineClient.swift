import Foundation
import Darwin

public enum EngineClientError: Error, LocalizedError {
    case notRunning
    case timedOut(String)
    case engineExited

    public var errorDescription: String? {
        switch self {
        case .notRunning: return L10n.text("The timer engine is not running.")
        case .timedOut(let method): return L10n.text("The timer engine did not answer ({0}).", method)
        case .engineExited: return L10n.text("The timer engine stopped unexpectedly.")
        }
    }
}

/// Runs the Go engine sidecar and talks JSON-RPC to it over stdio. A crashed
/// engine is restarted with backoff; closing stdin on `stop()` lets the engine
/// shut down cleanly.
public final class EngineClient: @unchecked Sendable {
    public struct Configuration {
        public var executableURL: URL
        public var arguments: [String]
        public var environment: [String: String]
        public var restartDelay: (Int) -> TimeInterval

        public init(executableURL: URL, arguments: [String], environment: [String: String],
                    restartDelay: @escaping (Int) -> TimeInterval = engineRestartDelay) {
            self.executableURL = executableURL; self.arguments = arguments
            self.environment = environment; self.restartDelay = restartDelay
        }
    }

    /// Callbacks run on the main queue.
    public var onEvent: ((EngineEvent) -> Void)?
    public var onStatus: ((EngineStatus) -> Void)?
    public var onRunningChange: ((Bool) -> Void)?
    public var onAgentDelta: ((String, String) -> Void)?

    private let configuration: Configuration
    private let queue = DispatchQueue(label: "break-reminder.engine-client")
    private var process: Process?
    private var input: FileHandle?
    private var buffer = LineBuffer()
    private var nextID = 1
    private var pending: [Int: CheckedContinuation<Data, Error>] = [:]
    private var restartAttempt = 0
    private var launchedAt = Date.distantPast
    private var stopping = true
    private var _launchCount = 0
    private var stderrTail = ""

    public init(configuration: Configuration) { self.configuration = configuration }

    public var launchCount: Int { queue.sync { _launchCount } }
    public var isRunning: Bool { queue.sync { process?.isRunning ?? false } }
    /// Last lines the engine wrote to stderr, for diagnostics.
    public var lastErrorOutput: String { queue.sync { stderrTail } }

    public func start() {
        // Writing to a dead engine's stdin must fail with EPIPE, not kill us.
        signal(SIGPIPE, SIG_IGN)
        queue.async {
            self.stopping = false
            self.launch()
        }
    }

    public func stop() {
        let proc: Process? = queue.sync {
            stopping = true
            let p = process
            try? input?.close()
            input = nil
            return p
        }
        guard let proc, proc.isRunning else { return }
        // Give the engine a moment to finish writing state, then make sure.
        let deadline = Date().addingTimeInterval(2)
        while proc.isRunning && Date() < deadline { usleep(20_000) }
        if proc.isRunning { proc.terminate() }
    }

    public func call(_ method: String, params: [String: Any] = [:], timeout: TimeInterval = 5) async throws -> Data {
        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Data, Error>) in
            queue.async {
                guard let input = self.input, self.process?.isRunning == true else {
                    continuation.resume(throwing: EngineClientError.notRunning)
                    return
                }
                let id = self.nextID
                self.nextID += 1
                let line: Data
                do { line = try EngineRequest.encode(id: id, method: method, params: params) } catch {
                    continuation.resume(throwing: error)
                    return
                }
                self.pending[id] = continuation
                do { try input.write(contentsOf: line) } catch {
                    self.pending.removeValue(forKey: id)?.resume(throwing: EngineClientError.engineExited)
                    return
                }
                self.queue.asyncAfter(deadline: .now() + timeout) {
                    self.pending.removeValue(forKey: id)?.resume(throwing: EngineClientError.timedOut(method))
                }
            }
        }
    }

    public func call<T: Decodable>(_ method: String, params: [String: Any] = [:], as type: T.Type, timeout: TimeInterval = 5) async throws -> T {
        try JSONDecoder().decode(T.self, from: try await call(method, params: params, timeout: timeout))
    }

    // MARK: Process lifecycle (queue only)

    private func launch() {
        guard !stopping, process?.isRunning != true else { return }
        let proc = Process()
        proc.executableURL = configuration.executableURL
        proc.arguments = configuration.arguments
        proc.environment = configuration.environment
        let stdinPipe = Pipe(), stdoutPipe = Pipe(), stderrPipe = Pipe()
        proc.standardInput = stdinPipe
        proc.standardOutput = stdoutPipe
        proc.standardError = stderrPipe
        buffer = LineBuffer()

        stdoutPipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let chunk = handle.availableData
            guard let self else { return }
            if chunk.isEmpty { handle.readabilityHandler = nil; return }
            self.queue.async { self.receive(chunk) }
        }
        stderrPipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let chunk = handle.availableData
            guard let self else { return }
            if chunk.isEmpty { handle.readabilityHandler = nil; return }
            self.queue.async {
                self.stderrTail = String((self.stderrTail + String(decoding: chunk, as: UTF8.self)).suffix(2000))
            }
        }
        proc.terminationHandler = { [weak self] finished in
            guard let self else { return }
            self.queue.async { self.handleExit(finished) }
        }
        do {
            try proc.run()
        } catch {
            stderrTail = error.localizedDescription
            scheduleRestart()
            return
        }
        process = proc
        input = stdinPipe.fileHandleForWriting
        launchedAt = Date()
        _launchCount += 1
        notifyRunning(true)
    }

    private func receive(_ chunk: Data) {
        for line in buffer.append(chunk) {
            guard let inbound = try? EngineInbound.parse(line) else { continue }
            switch inbound {
            case let .response(id, result, error):
                guard let continuation = pending.removeValue(forKey: id) else { continue }
                if let error { continuation.resume(throwing: error) } else { continuation.resume(returning: result ?? Data("{}".utf8)) }
            case let .event(event):
                DispatchQueue.main.async { self.onEvent?(event) }
            case let .status(status):
                DispatchQueue.main.async { self.onStatus?(status) }
            case let .agentDelta(requestID, text):
                DispatchQueue.main.async { self.onAgentDelta?(requestID, text) }
            case .other:
                break
            }
        }
    }

    private func handleExit(_ finished: Process) {
        guard finished === process else { return }
        process = nil
        input = nil
        for (_, continuation) in pending { continuation.resume(throwing: EngineClientError.engineExited) }
        pending.removeAll()
        notifyRunning(false)
        if !stopping { scheduleRestart() }
    }

    private func scheduleRestart() {
        // An engine that stayed up for a while starts the backoff over.
        if Date().timeIntervalSince(launchedAt) > 60 { restartAttempt = 0 }
        let delay = configuration.restartDelay(restartAttempt)
        restartAttempt += 1
        queue.asyncAfter(deadline: .now() + delay) { self.launch() }
    }

    private func notifyRunning(_ running: Bool) {
        DispatchQueue.main.async { self.onRunningChange?(running) }
    }
}
