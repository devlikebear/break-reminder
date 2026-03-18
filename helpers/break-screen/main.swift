import AppKit
import Foundation

// MARK: - Command-line arguments

struct Args {
    var duration: Int = 600    // break duration in seconds
    var skipAfter: Int = 120   // seconds before skip button activates
}

func parseArgs() -> Args {
    var args = Args()
    let argv = CommandLine.arguments
    var i = 1
    while i < argv.count {
        switch argv[i] {
        case "--duration":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.duration = v }
        case "--skip-after":
            i += 1
            if i < argv.count, let v = Int(argv[i]) { args.skipAfter = v }
        default:
            break
        }
        i += 1
    }
    return args
}

// MARK: - App Delegate

class BreakScreenApp: NSObject, NSApplicationDelegate {
    var windows: [NSWindow] = []
    var countdownLabel: NSTextField!
    var progressView: NSView!
    var progressFill: NSView!
    var skipButton: NSButton!
    var activityLabel: NSTextField!

    let args: Args
    var remaining: Int
    var timer: Timer?
    var elapsed: Int = 0

    let activities = [
        "👁  Look at something 20 feet away for 20 seconds",
        "🤸 Stand up and stretch your neck and shoulders",
        "🌬  Take 5 deep breaths: inhale 4s, hold 4s, exhale 4s",
        "🚶 Take a short walk and get some water",
        "🧘 Close your eyes and relax your mind",
    ]

    init(args: Args) {
        self.args = args
        self.remaining = args.duration
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory) // No dock icon

        for screen in NSScreen.screens {
            createWindow(on: screen, isPrimary: screen == NSScreen.main)
        }

        NSApp.activate(ignoringOtherApps: true)

        // Global Esc key monitor — always available as emergency exit
        NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            if event.keyCode == 53 { // Esc key
                self?.quit()
                return nil
            }
            return event
        }

        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.tick()
        }
    }

    func createWindow(on screen: NSScreen, isPrimary: Bool) {
        let window = NSWindow(
            contentRect: screen.frame,
            styleMask: .borderless,
            backing: .buffered,
            defer: false,
            screen: screen
        )
        window.level = .screenSaver
        window.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        window.isOpaque = false
        window.backgroundColor = NSColor(white: 0.08, alpha: 0.95)
        window.ignoresMouseEvents = !isPrimary

        if isPrimary {
            let contentView = NSView(frame: screen.frame)
            setupPrimaryUI(in: contentView, frame: screen.frame)
            window.contentView = contentView
        } else {
            // Secondary screens just show a dark overlay with a small label
            let contentView = NSView(frame: screen.frame)
            let label = NSTextField(labelWithString: "☕ Break Time")
            label.font = NSFont.systemFont(ofSize: 36, weight: .light)
            label.textColor = NSColor(white: 0.5, alpha: 1.0)
            label.alignment = .center
            label.sizeToFit()
            label.frame.origin = NSPoint(
                x: (screen.frame.width - label.frame.width) / 2,
                y: screen.frame.height / 2 - label.frame.height / 2
            )
            contentView.addSubview(label)
            window.contentView = contentView
        }

        window.orderFrontRegardless()
        windows.append(window)
    }

    func setupPrimaryUI(in view: NSView, frame: NSRect) {
        let centerX = frame.width / 2
        let centerY = frame.height / 2

        // Title
        let title = NSTextField(labelWithString: "☕ Time for a Break!")
        title.font = NSFont.systemFont(ofSize: 48, weight: .bold)
        title.textColor = .white
        title.alignment = .center
        title.sizeToFit()
        title.frame.origin = NSPoint(x: centerX - title.frame.width / 2, y: centerY + 120)
        view.addSubview(title)

        // Countdown
        countdownLabel = NSTextField(labelWithString: formatTime(remaining))
        countdownLabel.font = NSFont.monospacedDigitSystemFont(ofSize: 96, weight: .ultraLight)
        countdownLabel.textColor = NSColor(red: 0.4, green: 0.8, blue: 1.0, alpha: 1.0)
        countdownLabel.alignment = .center
        countdownLabel.sizeToFit()
        countdownLabel.frame = NSRect(x: centerX - 150, y: centerY - 10, width: 300, height: 110)
        view.addSubview(countdownLabel)

        // Progress bar background
        let barWidth: CGFloat = 400
        let barHeight: CGFloat = 8
        progressView = NSView(frame: NSRect(
            x: centerX - barWidth / 2,
            y: centerY - 50,
            width: barWidth,
            height: barHeight
        ))
        progressView.wantsLayer = true
        progressView.layer?.backgroundColor = NSColor(white: 0.3, alpha: 1.0).cgColor
        progressView.layer?.cornerRadius = barHeight / 2
        view.addSubview(progressView)

        // Progress bar fill
        progressFill = NSView(frame: NSRect(x: 0, y: 0, width: 0, height: barHeight))
        progressFill.wantsLayer = true
        progressFill.layer?.backgroundColor = NSColor(red: 0.4, green: 0.8, blue: 1.0, alpha: 1.0).cgColor
        progressFill.layer?.cornerRadius = barHeight / 2
        progressView.addSubview(progressFill)

        // Activity suggestion
        let activity = activities.randomElement() ?? activities[0]
        activityLabel = NSTextField(labelWithString: activity)
        activityLabel.font = NSFont.systemFont(ofSize: 22, weight: .regular)
        activityLabel.textColor = NSColor(white: 0.7, alpha: 1.0)
        activityLabel.alignment = .center
        activityLabel.sizeToFit()
        activityLabel.frame = NSRect(
            x: centerX - 300,
            y: centerY - 120,
            width: 600,
            height: 30
        )
        view.addSubview(activityLabel)

        // Skip button (initially disabled)
        skipButton = NSButton(title: "Skip (available in \(args.skipAfter / 60)min)", target: self, action: #selector(skipBreak))
        skipButton.bezelStyle = .rounded
        skipButton.font = NSFont.systemFont(ofSize: 16)
        skipButton.isEnabled = false
        skipButton.sizeToFit()
        skipButton.frame.origin = NSPoint(
            x: centerX - skipButton.frame.width / 2,
            y: centerY - 200
        )
        // Style for dark background
        skipButton.contentTintColor = NSColor(white: 0.5, alpha: 1.0)
        view.addSubview(skipButton)

        // Esc hint
        let escHint = NSTextField(labelWithString: "Press Esc to dismiss anytime")
        escHint.font = NSFont.systemFont(ofSize: 14, weight: .light)
        escHint.textColor = NSColor(white: 0.35, alpha: 1.0)
        escHint.alignment = .center
        escHint.sizeToFit()
        escHint.frame.origin = NSPoint(
            x: centerX - escHint.frame.width / 2,
            y: centerY - 240
        )
        view.addSubview(escHint)
    }

    func tick() {
        elapsed += 1
        remaining -= 1

        if remaining <= 0 {
            quit()
            return
        }

        // Update countdown
        countdownLabel.stringValue = formatTime(remaining)

        // Update progress bar
        let progress = CGFloat(elapsed) / CGFloat(args.duration)
        let barWidth = progressView.frame.width
        progressFill.frame = NSRect(x: 0, y: 0, width: barWidth * progress, height: progressView.frame.height)

        // Enable skip button after skipAfter seconds
        if elapsed >= args.skipAfter && !skipButton.isEnabled {
            skipButton.isEnabled = true
            skipButton.title = "Skip Break"
            skipButton.contentTintColor = .white
        }
    }

    @objc func skipBreak() {
        quit()
    }

    func quit() {
        timer?.invalidate()
        for w in windows {
            w.orderOut(nil)
        }
        NSApp.terminate(nil)
    }

    func formatTime(_ seconds: Int) -> String {
        let m = seconds / 60
        let s = seconds % 60
        return String(format: "%02d:%02d", m, s)
    }
}

// MARK: - Main

let args = parseArgs()
let app = NSApplication.shared
let delegate = BreakScreenApp(args: args)
app.delegate = delegate
app.run()
