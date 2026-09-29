import Foundation
import Darwin

public struct CommandResult: Sendable {
    public var exitCode: Int32
    public var stdout: String
    public var stderr: String
    public var timedOut: Bool
    public var succeeded: Bool { exitCode == 0 && !timedOut }
    public init(exitCode: Int32, stdout: String = "", stderr: String = "", timedOut: Bool = false) {
        self.exitCode = exitCode; self.stdout = stdout; self.stderr = stderr; self.timedOut = timedOut
    }
    public var errorMessage: String {
        if timedOut { return "명령 응답 시간이 초과되었습니다. 상태를 다시 확인해 주세요." }
        return stderr.isEmpty ? "명령 실행 실패 (\(exitCode))" : stderr.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

public protocol CLICommandRunning {
    func run(arguments: [String]) async -> CommandResult
}

public struct CLICommandClient: CLICommandRunning, Sendable {
    public let executablePath: String?
    public let timeout: TimeInterval
    public init(executablePath: String?, timeout: TimeInterval = 10) {
        self.executablePath = executablePath; self.timeout = timeout
    }
    public func run(arguments: [String]) async -> CommandResult {
        await withCheckedContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                continuation.resume(returning: execute(arguments: arguments))
            }
        }
    }
    private func execute(arguments: [String]) -> CommandResult {
        guard let path = executablePath else {
            return CommandResult(exitCode: -1, stderr: "break-reminder CLI를 찾을 수 없습니다. 설치를 확인해 주세요.")
        }
        let process = Process()
        process.executableURL = URL(fileURLWithPath: path)
        process.arguments = arguments
        let output = Pipe(), errors = Pipe()
        process.standardOutput = output; process.standardError = errors
        let done = DispatchSemaphore(value: 0)
        process.terminationHandler = { _ in done.signal() }
        do { try process.run() } catch {
            return CommandResult(exitCode: -1, stderr: error.localizedDescription)
        }
        // Drain both pipes concurrently so a full stderr cannot deadlock stdout.
        let group = DispatchGroup()
        let outBox = OutputBox(), errBox = OutputBox()
        for (handle, box) in [(output.fileHandleForReading, outBox), (errors.fileHandleForReading, errBox)] {
            group.enter()
            DispatchQueue.global(qos: .utility).async {
                defer { group.leave() }
                var data = Data()
                while let chunk = try? handle.read(upToCount: 8192), !chunk.isEmpty {
                    if data.count < 1_048_576 { data.append(chunk.prefix(1_048_576 - data.count)) }
                }
                box.value = String(decoding: data, as: UTF8.self)
            }
        }
        let expired = done.wait(timeout: .now() + timeout) == .timedOut
        if expired {
            process.terminate()
            if done.wait(timeout: .now() + 0.5) == .timedOut {
                kill(process.processIdentifier, SIGKILL)
                done.wait()
            }
        }
        group.wait()
        return CommandResult(exitCode: process.terminationStatus, stdout: outBox.value, stderr: errBox.value, timedOut: expired)
    }
}

private final class OutputBox: @unchecked Sendable {
    // One writer per box, read only after DispatchGroup.wait().
    var value = ""
}
