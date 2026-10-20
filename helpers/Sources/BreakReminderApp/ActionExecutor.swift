import AppKit
import AVFoundation
import CoreGraphics
import HelperCore
import UserNotifications

/// Executes engine events with native APIs: notifications, speech, the break
/// screen overlay, and the "ask" prompt. Nothing here decides *when* — the
/// engine already did.
@MainActor
final class ActionExecutor {
    private let model: AppModel
    private let speech = AVSpeechSynthesizer()
    private var breakScreen: Process?
    private var notificationsAuthorized = false

    init(model: AppModel) {
        self.model = model
    }

    /// Notification permission needs a real bundle (not a bare `swift run`).
    private var canUseUserNotifications: Bool { Bundle.main.bundleIdentifier != nil }

    func requestNotificationPermission() {
        guard canUseUserNotifications else { return }
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { granted, _ in
            Task { @MainActor in self.notificationsAuthorized = granted }
        }
    }

    func handle(_ event: EngineEvent) {
        switch event.type {
        case "break_start":
            startBreak(event)
        case "speak":
            speak(event.text ?? "", voiceName: event.voice ?? "")
        default:
            notify(event)
        }
    }

    // MARK: Notifications

    private func notify(_ event: EngineEvent) {
        guard let title = event.title, !title.isEmpty else { return }
        if model.status?.notificationsEnabled == false { return }
        if let sound = event.sound, !sound.isEmpty { NSSound(named: NSSound.Name(sound))?.play() }
        guard canUseUserNotifications else { return }
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = event.body ?? ""
        let id = event.id ?? "\(event.type)-\(UUID().uuidString)"
        UNUserNotificationCenter.current().add(UNNotificationRequest(identifier: id, content: content, trigger: nil))
    }

    // MARK: Speech

    private func speak(_ text: String, voiceName: String) {
        guard !text.isEmpty else { return }
        let sandboxed = ProcessInfo.processInfo.environment["APP_SANDBOX_CONTAINER_ID"] != nil
        if speechRoute(ttsEngine: model.status?.ttsEngine, sandboxed: sandboxed) == .engineCLI,
           let engine = BundledTool.engine {
            let process = Process()
            process.executableURL = engine
            process.arguments = ["tts", "test", text]
            process.environment = engineEnvironment()
            process.standardOutput = FileHandle.nullDevice
            process.standardError = FileHandle.nullDevice
            process.terminationHandler = { finished in
                guard finished.terminationStatus != 0 else { return }
                Task { @MainActor in self.speakWithSystemVoice(text, voiceName: voiceName) }
            }
            if (try? process.run()) != nil { return }
        }
        speakWithSystemVoice(text, voiceName: voiceName)
    }

    private func speakWithSystemVoice(_ text: String, voiceName: String) {
        let utterance = AVSpeechUtterance(string: text)
        let voices = AVSpeechSynthesisVoice.speechVoices().map {
            SpeechVoiceInfo(identifier: $0.identifier, name: $0.name, language: $0.language)
        }
        if let id = matchSpeechVoice(named: voiceName, in: voices, fallbackLanguage: AppLanguage.current.rawValue) {
            utterance.voice = AVSpeechSynthesisVoice(identifier: id)
        }
        speech.speak(utterance)
    }

    // MARK: Break screen

    private func startBreak(_ event: EngineEvent) {
        switch event.screenMode ?? "notify" {
        case "block":
            showOverlay(event)
        case "ask":
            askBreakMode(event)
        default:
            notify(event)
        }
    }

    private func askBreakMode(_ event: EngineEvent) {
        NSApp.activate(ignoringOtherApps: true)
        let alert = NSAlert()
        alert.messageText = L10n.text("Break Time! How would you like to be reminded?")
        alert.informativeText = L10n.text("• Block Screen: Full-screen overlay until break ends") + "\n" + L10n.text("• Notification Only: Just show notifications")
        alert.addButton(withTitle: L10n.text("Block Screen"))
        alert.addButton(withTitle: L10n.text("Notification Only"))
        let block = alert.runModal() == .alertFirstButtonReturn
        model.perform("break_screen.choose", ["mode": block ? "block" : "notify"])
        if block { showOverlay(event) } else { notify(event) }
    }

    private func showOverlay(_ event: EngineEvent) {
        if breakScreen?.isRunning == true { return }
        guard let helper = BundledTool.breakScreen,
              let args = breakScreenArguments(event: event, now: Int64(Date().timeIntervalSince1970)) else {
            notify(event)
            return
        }
        let process = Process()
        process.executableURL = helper
        process.arguments = args
        var env = ProcessInfo.processInfo.environment
        env["BREAK_REMINDER_LANGUAGE"] = AppLanguage.current.rawValue
        process.environment = env
        do {
            try process.run()
            breakScreen = process
        } catch {
            notify(event)
        }
    }

    func stopOverlay() {
        if breakScreen?.isRunning == true { breakScreen?.terminate() }
    }
}

/// Measures system idle time and reports it to the engine.
@MainActor
final class IdleReporter {
    private var timer: Timer?

    static func idleSeconds() -> Int {
        // Development hook: pin idle time to exercise the cycle unattended.
        if let fake = ProcessInfo.processInfo.environment["BREAK_REMINDER_FAKE_IDLE"], let value = Int(fake) {
            return value
        }
        let anyInput = CGEventType(rawValue: UInt32.max)!
        let seconds = CGEventSource.secondsSinceLastEventType(.combinedSessionState, eventType: anyInput)
        return seconds.isFinite ? Int(seconds) : 0
    }

    func start(every interval: TimeInterval = 10, report: @escaping (Int) -> Void) {
        report(Self.idleSeconds())
        timer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { _ in
            Task { @MainActor in report(IdleReporter.idleSeconds()) }
        }
    }
}
