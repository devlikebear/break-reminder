import Foundation
import Combine
import HelperCore

@MainActor
final class TimeToolsViewModel: ObservableObject {
    @Published private(set) var snapshot = TimeToolsSnapshot()
    @Published private(set) var runtime = TimeToolsRuntime()
    @Published private(set) var nowMS = Int64(Date().timeIntervalSince1970 * 1000)
    @Published private(set) var isBusy = false
    @Published private(set) var readError: String?
    @Published private(set) var commandError: String?
    let files = TimeToolsFiles()
    private var timer: Timer?
    var workerRunning: Bool { runtime.isRunning(nowMS: nowMS) }

    func startRefreshing() {
        refresh()
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.refresh() }
        }
    }
    func stopRefreshing() { timer?.invalidate(); timer = nil }
    func refresh() {
        nowMS = Int64(Date().timeIntervalSince1970 * 1000)
        runtime = files.loadRuntime()
        do { snapshot = try files.loadSnapshot(); readError = nil }
        catch { readError = "시간 도구 파일을 읽지 못했습니다: \(error.localizedDescription)" }
    }
    func send(_ command: [String]) {
        guard !isBusy, readError == nil else { return }
        execute(files.arguments(command + ["--if-revision", String(snapshot.revision)]))
    }
    func startTimer(seconds: Int, label: String, replaceID: String? = nil) {
        var command = ["timer", "start", "--duration", "\(seconds)s", "--label", label]
        if let replaceID { command += ["--replace-id", replaceID] }
        send(command)
    }
    func setupService() { execute(["service", "install"]) }
    private func execute(_ args: [String]) {
        guard !isBusy else { return }
        isBusy = true; commandError = nil
        Task { @MainActor in
            let result = await CLICommandClient(executablePath: findHelper("break-reminder")).run(arguments: args)
            if !result.succeeded { commandError = result.errorMessage }
            isBusy = false; refresh()
        }
    }
}
