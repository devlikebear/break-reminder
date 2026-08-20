import AppKit
import Foundation
import HelperCore

// MARK: - Key-accepting borderless window

class KeyWindow: NSWindow {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

// MARK: - App Delegate

class BreakScreenApp: NSObject, NSApplicationDelegate {
    var windows: [NSWindow] = []
    var primaryWindow: NSWindow?
    var countdownLabel: NSTextField!
    var progressView: NSView!
    var progressFill: NSView!
    var skipButton: NSButton!
    var guideCardView: NSView!
    var guideEyebrowLabel: NSTextField!
    var guideTitleLabel: NSTextField!
    var guideCountdownLabel: NSTextField!
    var guideInstructionLabel: NSTextField!
    var guideStatusLabel: NSTextField!
    var guideActionButton: NSButton!

    let args: BreakScreenArgs
    var remaining: Int
    var timer: Timer?
    var elapsed: Int = 0
    var guidedSession = GuidedBreakSession()

    private var completionAnnounced = false

    init(args: BreakScreenArgs) {
        self.args = args
        self.remaining = args.duration
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        let mainScreen = NSScreen.main ?? NSScreen.screens.first

        for screen in NSScreen.screens {
            createWindow(on: screen, isPrimary: screen === mainScreen)
        }

        DispatchQueue.main.async {
            NSApp.activate(ignoringOtherApps: true)
            for window in self.windows {
                window.orderFrontRegardless()
            }
            self.primaryWindow?.makeKeyAndOrderFront(nil)
            self.renderGuidedSession(focusAction: true)
        }

        NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            if event.keyCode == 53 {
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
        let window = KeyWindow(
            contentRect: screen.frame,
            styleMask: .borderless,
            backing: .buffered,
            defer: false,
            screen: screen
        )
        window.level = .screenSaver
        window.collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle, .fullScreenAuxiliary]
        window.isOpaque = false
        window.backgroundColor = NSColor(white: 0.08, alpha: 0.95)
        window.ignoresMouseEvents = !isPrimary
        window.setFrame(screen.frame, display: false)

        let localFrame = NSRect(origin: .zero, size: screen.frame.size)

        if isPrimary {
            let contentView = NSView(frame: localFrame)
            setupPrimaryUI(in: contentView, frame: localFrame)
            window.contentView = contentView
        } else {
            let contentView = NSView(frame: localFrame)
            let label = NSTextField(labelWithString: "☕ Break Time")
            label.font = NSFont.systemFont(ofSize: 36, weight: .light)
            label.textColor = NSColor(white: 0.5, alpha: 1.0)
            label.alignment = .center
            label.sizeToFit()
            label.frame.origin = NSPoint(
                x: (localFrame.width - label.frame.width) / 2,
                y: localFrame.height / 2 - label.frame.height / 2
            )
            contentView.addSubview(label)
            window.contentView = contentView
        }

        if isPrimary {
            primaryWindow = window
        }
        windows.append(window)
    }

    func setupPrimaryUI(in view: NSView, frame: NSRect) {
        let centerX = frame.width / 2
        let compactHeight = frame.height < 700
        let contentWidth = min(600, max(0, frame.width - 48))
        let progressWidth = min(400, contentWidth)
        let titleHeight: CGFloat = compactHeight ? 48 : 58
        let countdownHeight: CGFloat = compactHeight ? 92 : 110
        let titleGap: CGFloat = compactHeight ? 8 : 16
        let cardGap: CGFloat = compactHeight ? 16 : 24
        let statsVisible = args.todayWorkMin > 0 || args.todayBreakMin > 0
        let statsBlockHeight: CGFloat = statsVisible ? (compactHeight ? 32 : 40) : 0
        let lowerGap: CGFloat = compactHeight ? 8 : 16
        let totalHeight = titleHeight + titleGap + countdownHeight + 8 + 8 + cardGap
            + 196 + statsBlockHeight + lowerGap + 44 + 8 + 18
        var top = max(24, (frame.height - totalHeight) / 2) + totalHeight

        let title = NSTextField(labelWithString: "휴식 시간이에요")
        title.font = NSFont.systemFont(ofSize: compactHeight ? 40 : 48, weight: .bold)
        title.textColor = .white
        title.alignment = .center
        title.frame = NSRect(x: centerX - contentWidth / 2, y: top - titleHeight, width: contentWidth, height: titleHeight)
        title.setAccessibilityLabel("휴식 시간이에요")
        view.addSubview(title)
        top -= titleHeight + titleGap

        countdownLabel = NSTextField(labelWithString: formatTime(remaining))
        countdownLabel.font = NSFont.monospacedDigitSystemFont(
            ofSize: compactHeight ? 80 : 96,
            weight: .ultraLight
        )
        countdownLabel.textColor = NSColor(red: 0.4, green: 0.702, blue: 1.0, alpha: 1.0)
        countdownLabel.alignment = .center
        countdownLabel.frame = NSRect(
            x: centerX - contentWidth / 2,
            y: top - countdownHeight,
            width: contentWidth,
            height: countdownHeight
        )
        countdownLabel.setAccessibilityLabel("전체 휴식 남은 시간")
        countdownLabel.setAccessibilityValue(accessibilityDuration(remaining))
        view.addSubview(countdownLabel)
        top -= countdownHeight + 8

        let barHeight: CGFloat = 8
        progressView = NSView(frame: NSRect(
            x: centerX - progressWidth / 2,
            y: top - barHeight,
            width: progressWidth,
            height: barHeight
        ))
        progressView.wantsLayer = true
        progressView.layer?.backgroundColor = NSColor(white: 0.3, alpha: 1.0).cgColor
        progressView.layer?.cornerRadius = barHeight / 2
        progressView.setAccessibilityElement(true)
        progressView.setAccessibilityRole(.progressIndicator)
        progressView.setAccessibilityLabel("전체 휴식 진행률")
        progressView.setAccessibilityValue("0퍼센트")
        view.addSubview(progressView)

        progressFill = NSView(frame: NSRect(x: 0, y: 0, width: 0, height: barHeight))
        progressFill.wantsLayer = true
        progressFill.layer?.backgroundColor = NSColor(red: 0.4, green: 0.702, blue: 1.0, alpha: 1.0).cgColor
        progressFill.layer?.cornerRadius = barHeight / 2
        progressFill.setAccessibilityElement(false)
        progressView.addSubview(progressFill)
        top -= barHeight + cardGap

        guideCardView = NSView(frame: NSRect(
            x: centerX - contentWidth / 2,
            y: top - 196,
            width: contentWidth,
            height: 196
        ))
        guideCardView.wantsLayer = true
        guideCardView.layer?.backgroundColor = NSColor(white: 0.145, alpha: 0.96).cgColor
        guideCardView.layer?.cornerRadius = 16
        guideCardView.setAccessibilityElement(true)
        guideCardView.setAccessibilityRole(.group)
        view.addSubview(guideCardView)

        guideEyebrowLabel = makeCardLabel(fontSize: 13, weight: .semibold)
        guideEyebrowLabel.textColor = NSColor(white: 0.7, alpha: 1.0)
        guideCardView.addSubview(guideEyebrowLabel)

        guideTitleLabel = makeCardLabel(fontSize: 22, weight: .semibold)
        guideCardView.addSubview(guideTitleLabel)

        guideCountdownLabel = makeCardLabel(fontSize: 48, weight: .light, monospacedDigits: true)
        guideCardView.addSubview(guideCountdownLabel)

        guideInstructionLabel = makeCardLabel(fontSize: 18, weight: .regular)
        guideInstructionLabel.textColor = NSColor(white: 0.7, alpha: 1.0)
        guideCardView.addSubview(guideInstructionLabel)

        guideStatusLabel = makeCardLabel(fontSize: 14, weight: .regular)
        guideStatusLabel.textColor = NSColor(white: 0.5, alpha: 1.0)
        guideCardView.addSubview(guideStatusLabel)

        guideActionButton = NSButton(title: "시작", target: self, action: #selector(startGuidedBreak))
        guideActionButton.bezelStyle = .rounded
        guideActionButton.font = NSFont.systemFont(ofSize: 16, weight: .medium)
        guideActionButton.frame = NSRect(x: (contentWidth - 120) / 2, y: 14, width: 120, height: 44)
        guideCardView.addSubview(guideActionButton)
        top -= 196

        if statsVisible {
            top -= compactHeight ? 8 : 16
            let statsText = "오늘: 작업 \(formatMinutes(args.todayWorkMin)) · 휴식 \(formatMinutes(args.todayBreakMin))"
            let statsLabel = NSTextField(labelWithString: statsText)
            statsLabel.font = NSFont.systemFont(ofSize: 16, weight: .medium)
            statsLabel.textColor = NSColor(white: 0.5, alpha: 1.0)
            statsLabel.alignment = .center
            statsLabel.frame = NSRect(x: centerX - contentWidth / 2, y: top - 24, width: contentWidth, height: 24)
            view.addSubview(statsLabel)
            top -= 24
        }

        top -= lowerGap
        skipButton = NSButton(
            title: "Skip (available in \(args.skipAfter / 60)min)",
            target: self,
            action: #selector(skipBreak)
        )
        skipButton.bezelStyle = .rounded
        skipButton.font = NSFont.systemFont(ofSize: 16, weight: .medium)
        skipButton.isEnabled = false
        skipButton.contentTintColor = NSColor(white: 0.5, alpha: 1.0)
        skipButton.setAccessibilityLabel("휴식 건너뛰기")
        skipButton.setAccessibilityHelp("현재 휴식 화면을 닫습니다.")
        let skipWidth = max(120, skipButton.fittingSize.width + 24)
        skipButton.frame = NSRect(x: centerX - skipWidth / 2, y: top - 44, width: skipWidth, height: 44)
        view.addSubview(skipButton)
        top -= 52

        let escHint = NSTextField(labelWithString: "Esc를 누르면 언제든 닫혀요")
        escHint.font = NSFont.systemFont(ofSize: 14, weight: .light)
        escHint.textColor = NSColor(white: 0.35, alpha: 1.0)
        escHint.alignment = .center
        escHint.frame = NSRect(x: centerX - contentWidth / 2, y: top - 18, width: contentWidth, height: 18)
        view.addSubview(escHint)

        renderGuidedSession()
    }

    func tick() {
        elapsed += 1
        remaining -= 1

        // The overall break owns the overlay lifetime and always wins a same-tick race.
        if remaining <= 0 {
            quit()
            return
        }

        let skipBecameEnabled = updateOverallBreakUI()

        switch guidedSession.tick() {
        case .stay:
            let shouldFocusSkip = skipBecameEnabled && shouldPreferSkipFocus
            renderGuidedSession(focusAction: shouldFocusSkip)
        case .phaseChanged:
            renderGuidedSession(focusAction: true)
            if case .completed = guidedSession.phase {
                announceCompletion()
            }
        case .dismiss:
            quit()
        }
    }

    @objc func startGuidedBreak() {
        guard guidedSession.start(availableBreakSeconds: remaining) else {
            renderGuidedSession(focusAction: true)
            return
        }

        completionAnnounced = false
        renderGuidedSession(focusAction: true)
    }

    @objc func cancelGuidedBreak() {
        guidedSession.cancel()
        renderGuidedSession(focusAction: true)
    }

    @objc func skipBreak() { quit() }

    func quit() {
        timer?.invalidate()
        for w in windows { w.orderOut(nil) }
        NSApp.terminate(nil)
    }

    private func makeCardLabel(
        fontSize: CGFloat,
        weight: NSFont.Weight,
        monospacedDigits: Bool = false
    ) -> NSTextField {
        let label = NSTextField(labelWithString: "")
        label.font = monospacedDigits
            ? NSFont.monospacedDigitSystemFont(ofSize: fontSize, weight: weight)
            : NSFont.systemFont(ofSize: fontSize, weight: weight)
        label.textColor = .white
        label.alignment = .center
        label.maximumNumberOfLines = 2
        label.lineBreakMode = .byWordWrapping
        label.cell?.wraps = true
        label.cell?.isScrollable = false
        return label
    }

    private func updateOverallBreakUI() -> Bool {
        countdownLabel.stringValue = formatTime(remaining)
        countdownLabel.setAccessibilityValue(accessibilityDuration(remaining))

        let duration = max(args.duration, 1)
        let progress = min(1, max(0, CGFloat(elapsed) / CGFloat(duration)))
        let barWidth = progressView.frame.width
        progressFill.frame = NSRect(
            x: 0,
            y: 0,
            width: barWidth * progress,
            height: progressView.frame.height
        )
        progressView.setAccessibilityValue("\(Int((progress * 100).rounded()))퍼센트")

        if elapsed >= args.skipAfter && !skipButton.isEnabled {
            skipButton.isEnabled = true
            skipButton.title = "Skip Break"
            skipButton.contentTintColor = .white
            return true
        }

        return false
    }

    private func renderGuidedSession(focusAction: Bool = false) {
        guard guideCardView != nil else { return }

        let cardWidth = guideCardView.bounds.width

        switch guidedSession.phase {
        case .ready:
            let canStart = remaining >= GuidedBreakSession.activityDurationSeconds
                + GuidedBreakSession.completionDisplaySeconds

            guideEyebrowLabel.isHidden = false
            guideEyebrowLabel.stringValue = "2분 가이드"
            guideEyebrowLabel.frame = NSRect(x: 24, y: 164, width: cardWidth - 48, height: 18)

            guideTitleLabel.stringValue = "2분 동안 서서 목과 어깨를 풀어보세요"
            guideTitleLabel.frame = NSRect(x: 24, y: 110, width: cardWidth - 48, height: 48)

            guideCountdownLabel.isHidden = true

            let instruction = canStart
                ? "같은 화면에서 천천히 따라 해요."
                : "이번 휴식에는 2분이 남지 않았어요."
            guideInstructionLabel.stringValue = instruction
            guideInstructionLabel.frame = NSRect(x: 24, y: 66, width: cardWidth - 48, height: 36)

            guideStatusLabel.isHidden = true

            guideActionButton.isHidden = false
            guideActionButton.isEnabled = canStart
            guideActionButton.title = "시작"
            guideActionButton.action = #selector(startGuidedBreak)
            guideActionButton.contentTintColor = canStart ? .white : NSColor(white: 0.5, alpha: 1.0)
            guideActionButton.frame = NSRect(x: (cardWidth - 120) / 2, y: 14, width: 120, height: 44)
            guideActionButton.setAccessibilityLabel("2분 목과 어깨 스트레칭 시작")
            guideActionButton.setAccessibilityHelp("같은 화면에서 2분 가이드를 시작합니다.")

            guideCardView.setAccessibilityLabel("2분 가이드")
            guideCardView.setAccessibilityValue("\(guideTitleLabel.stringValue). \(instruction)")

        case let .running(remainingSeconds):
            let instruction = guidedSession.instructionText()

            guideEyebrowLabel.isHidden = true

            guideTitleLabel.stringValue = "목과 어깨 스트레칭"
            guideTitleLabel.frame = NSRect(x: 24, y: 158, width: cardWidth - 48, height: 28)

            guideCountdownLabel.isHidden = false
            guideCountdownLabel.stringValue = formatTime(remainingSeconds)
            guideCountdownLabel.frame = NSRect(x: 24, y: 100, width: cardWidth - 48, height: 56)
            guideCountdownLabel.setAccessibilityLabel("가이드 남은 시간")
            guideCountdownLabel.setAccessibilityValue(accessibilityDuration(remainingSeconds))

            guideInstructionLabel.stringValue = instruction
            guideInstructionLabel.frame = NSRect(x: 24, y: 56, width: cardWidth - 48, height: 42)

            guideStatusLabel.isHidden = true

            guideActionButton.isHidden = false
            guideActionButton.isEnabled = true
            guideActionButton.title = "가이드 취소"
            guideActionButton.action = #selector(cancelGuidedBreak)
            guideActionButton.contentTintColor = .white
            guideActionButton.frame = NSRect(x: (cardWidth - 140) / 2, y: 8, width: 140, height: 44)
            guideActionButton.setAccessibilityLabel("가이드 취소")
            guideActionButton.setAccessibilityHelp("가이드를 멈추고 휴식 화면으로 돌아갑니다.")

            guideCardView.setAccessibilityLabel("목과 어깨 스트레칭")
            guideCardView.setAccessibilityValue(
                "남은 시간 \(accessibilityDuration(remainingSeconds)), \(instruction)"
            )

        case .completed:
            guideEyebrowLabel.isHidden = true

            guideTitleLabel.stringValue = "완료했어요"
            guideTitleLabel.frame = NSRect(x: 24, y: 142, width: cardWidth - 48, height: 32)

            guideCountdownLabel.isHidden = true

            guideInstructionLabel.stringValue = "편안하게 남은 휴식을 이어가세요."
            guideInstructionLabel.frame = NSRect(x: 24, y: 86, width: cardWidth - 48, height: 48)

            guideStatusLabel.isHidden = false
            guideStatusLabel.stringValue = "3초 후 이 화면을 닫아요."
            guideStatusLabel.frame = NSRect(x: 24, y: 54, width: cardWidth - 48, height: 22)

            guideActionButton.isHidden = true
            guideActionButton.isEnabled = false

            guideCardView.setAccessibilityLabel("완료했어요")
            guideCardView.setAccessibilityValue(guidedSession.instructionText())
            guideCardView.setAccessibilityHelp("3초 후 이 화면을 닫아요.")
        }

        updateKeyViewLoop(focusAction: focusAction)
    }

    private func updateKeyViewLoop(focusAction: Bool) {
        let guideIsFocusable = !guideActionButton.isHidden && guideActionButton.isEnabled
        let skipIsFocusable = skipButton.isEnabled
        let guideHadFocus = primaryWindow?.firstResponder === guideActionButton
        let startIsDefault: Bool
        if case .ready = guidedSession.phase {
            startIsDefault = guideIsFocusable
        } else {
            startIsDefault = false
        }

        guideActionButton.refusesFirstResponder = !guideIsFocusable
        skipButton.refusesFirstResponder = !skipIsFocusable
        guideActionButton.keyEquivalent = startIsDefault ? "\r" : ""
        primaryWindow?.defaultButtonCell = startIsDefault
            ? guideActionButton.cell as? NSButtonCell
            : nil

        guideActionButton.nextKeyView = skipIsFocusable ? skipButton : nil
        skipButton.nextKeyView = guideIsFocusable ? guideActionButton : nil

        guard focusAction || (guideHadFocus && !guideIsFocusable),
              let window = primaryWindow else {
            return
        }

        if guideIsFocusable {
            window.makeFirstResponder(guideActionButton)
        } else if skipIsFocusable {
            window.makeFirstResponder(skipButton)
        } else {
            window.makeFirstResponder(nil)
        }
    }

    private func announceCompletion() {
        guard !completionAnnounced,
              let element = primaryWindow?.contentView else {
            return
        }

        completionAnnounced = true
        NSAccessibility.post(
            element: element,
            notification: .announcementRequested,
            userInfo: [
                .announcement: "2분 스트레칭을 완료했습니다.",
                .priority: NSAccessibilityPriorityLevel.high.rawValue,
            ]
        )
    }

    private func accessibilityDuration(_ seconds: Int) -> String {
        let clampedSeconds = max(0, seconds)
        return "\(clampedSeconds / 60)분 \(clampedSeconds % 60)초"
    }

    private var shouldPreferSkipFocus: Bool {
        switch guidedSession.phase {
        case .ready:
            return remaining < GuidedBreakSession.activityDurationSeconds
                + GuidedBreakSession.completionDisplaySeconds
        case .running:
            return false
        case .completed:
            return true
        }
    }
}

// MARK: - Main

let args = parseBreakScreenArgs(CommandLine.arguments)
let app = NSApplication.shared
let delegate = BreakScreenApp(args: args)
app.delegate = delegate
app.run()
