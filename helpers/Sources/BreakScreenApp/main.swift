import AppKit
import Foundation
import HelperCore
import QuartzCore

// MARK: - Key-accepting borderless window

class KeyWindow: NSWindow {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

// MARK: - App Delegate

class BreakScreenApp: NSObject, NSApplicationDelegate {
    var windows: [NSWindow] = []
    var primaryWindow: NSWindow?
    var mascotImageView: NSImageView!
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
    var timeToolsCompletionLabel: NSTextField?
    private let timeToolsFiles = TimeToolsFiles()

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
            installBackdrop(in: contentView)
            setupPrimaryUI(in: contentView, frame: localFrame)
            window.contentView = contentView
        } else {
            let contentView = NSView(frame: localFrame)
            installBackdrop(in: contentView)
            setupSecondaryUI(in: contentView, frame: localFrame)
            window.contentView = contentView
        }

        if isPrimary {
            primaryWindow = window
        }
        windows.append(window)
    }

    private func installBackdrop(in view: NSView) {
        view.wantsLayer = true

        let gradient = CAGradientLayer()
        gradient.frame = view.bounds
        gradient.colors = [
            NSColor(srgbRed: 0.025, green: 0.055, blue: 0.12, alpha: 1.0).cgColor,
            NSColor(srgbRed: 0.045, green: 0.105, blue: 0.17, alpha: 1.0).cgColor,
            NSColor(srgbRed: 0.105, green: 0.065, blue: 0.17, alpha: 1.0).cgColor,
        ]
        gradient.startPoint = CGPoint(x: 0.08, y: 0.95)
        gradient.endPoint = CGPoint(x: 0.92, y: 0.05)
        view.layer?.insertSublayer(gradient, at: 0)
    }

    private func setupSecondaryUI(in view: NSView, frame: NSRect) {
        let centerX = frame.width / 2
        let compactHeight = frame.height < 700
        let contentWidth = min(600, max(0, frame.width - 48))
        let imageSide = min(compactHeight ? 140 : 200, contentWidth)

        if let image = loadMascotImage() {
            let imageView = makeMascotImageView(
                image: image,
                frame: NSRect(
                    x: centerX - imageSide / 2,
                    y: frame.height / 2 - imageSide / 2 + (compactHeight ? 24 : 36),
                    width: imageSide,
                    height: imageSide
                )
            )
            view.addSubview(imageView)
        }

        let title = NSTextField(labelWithString: L10n.text("Take a short break"))
        title.font = NSFont.systemFont(ofSize: compactHeight ? 28 : 36, weight: .semibold)
        title.textColor = NSColor(white: 0.94, alpha: 1.0)
        title.alignment = .center
        title.frame = NSRect(
            x: centerX - contentWidth / 2,
            y: frame.height / 2 - imageSide / 2 - (compactHeight ? 48 : 62),
            width: contentWidth,
            height: compactHeight ? 38 : 46
        )
        view.addSubview(title)

        let subtitle = NSTextField(labelWithString: L10n.text("☕ Break Time"))
        subtitle.font = NSFont.systemFont(ofSize: compactHeight ? 16 : 18, weight: .regular)
        subtitle.textColor = NSColor(white: 0.62, alpha: 1.0)
        subtitle.alignment = .center
        subtitle.frame = NSRect(
            x: centerX - contentWidth / 2,
            y: title.frame.minY - (compactHeight ? 24 : 28),
            width: contentWidth,
            height: 24
        )
        view.addSubview(subtitle)
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
        let mascotImage = loadMascotImage()
        let mascotSide: CGFloat = mascotImage == nil ? 0 : min(compactHeight ? 112 : 168, contentWidth)
        let mascotGap: CGFloat = mascotImage == nil ? 0 : (compactHeight ? 4 : 10)
        let totalHeight = mascotSide + mascotGap + titleHeight + titleGap + countdownHeight + 8 + 8 + cardGap
            + 196 + statsBlockHeight + lowerGap + 44 + 8 + 18
        var top = max(24, (frame.height - totalHeight) / 2) + totalHeight

        if let mascotImage {
            mascotImageView = makeMascotImageView(
                image: mascotImage,
                frame: NSRect(
                    x: centerX - mascotSide / 2,
                    y: top - mascotSide,
                    width: mascotSide,
                    height: mascotSide
                )
            )
            view.addSubview(mascotImageView)
            startMascotAnimation()
            top -= mascotSide + mascotGap
        }

        let title = NSTextField(labelWithString: L10n.text("Time for a break"))
        title.font = NSFont.systemFont(ofSize: compactHeight ? 40 : 48, weight: .bold)
        title.textColor = .white
        title.alignment = .center
        title.frame = NSRect(x: centerX - contentWidth / 2, y: top - titleHeight, width: contentWidth, height: titleHeight)
        title.setAccessibilityLabel(L10n.text("Time for a break"))
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
        countdownLabel.setAccessibilityLabel(L10n.text("Total break time remaining"))
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
        progressView.setAccessibilityLabel(L10n.text("Overall break progress"))
        progressView.setAccessibilityValue(L10n.text("0 percent"))
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
        guideCardView.layer?.backgroundColor = NSColor(white: 0.13, alpha: 0.94).cgColor
        guideCardView.layer?.cornerRadius = 16
        guideCardView.layer?.borderColor = NSColor.white.withAlphaComponent(0.08).cgColor
        guideCardView.layer?.borderWidth = 1
        guideCardView.layer?.shadowColor = NSColor.black.cgColor
        guideCardView.layer?.shadowOpacity = 0.28
        guideCardView.layer?.shadowRadius = 18
        guideCardView.layer?.shadowOffset = CGSize(width: 0, height: -8)
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

        guideActionButton = NSButton(title: L10n.text("Start"), target: self, action: #selector(startGuidedBreak))
        guideActionButton.bezelStyle = .rounded
        guideActionButton.font = NSFont.systemFont(ofSize: 16, weight: .medium)
        guideActionButton.frame = NSRect(x: (contentWidth - 120) / 2, y: 14, width: 120, height: 44)
        guideCardView.addSubview(guideActionButton)
        top -= 196

        if statsVisible {
            top -= compactHeight ? 8 : 16
            let statsText = L10n.text("Today · Work {0} · Break {1}", formatMinutes(args.todayWorkMin), formatMinutes(args.todayBreakMin))
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
            title: L10n.text("Skip (available in {0}min)", args.skipAfter / 60),
            target: self,
            action: #selector(skipBreak)
        )
        skipButton.bezelStyle = .rounded
        skipButton.font = NSFont.systemFont(ofSize: 16, weight: .medium)
        skipButton.isEnabled = false
        skipButton.contentTintColor = NSColor(white: 0.5, alpha: 1.0)
        skipButton.setAccessibilityLabel(L10n.text("Skip Break"))
        skipButton.setAccessibilityHelp(L10n.text("Closes the current break screen."))
        let skipWidth = max(120, skipButton.fittingSize.width + 24)
        skipButton.frame = NSRect(x: centerX - skipWidth / 2, y: top - 44, width: skipWidth, height: 44)
        view.addSubview(skipButton)
        top -= 52

        let escHint = NSTextField(labelWithString: L10n.text("Press Esc to close at any time"))
        escHint.font = NSFont.systemFont(ofSize: 14, weight: .light)
        escHint.textColor = NSColor(white: 0.35, alpha: 1.0)
        escHint.alignment = .center
        escHint.frame = NSRect(x: centerX - contentWidth / 2, y: top - 18, width: contentWidth, height: 18)
        view.addSubview(escHint)

        let toolsLabel = NSTextField(wrappingLabelWithString: "")
        toolsLabel.font = .systemFont(ofSize: 14, weight: .medium)
        toolsLabel.textColor = .systemOrange
        toolsLabel.alignment = .center
        toolsLabel.maximumNumberOfLines = 2
        toolsLabel.frame = NSRect(x: 24, y: 18, width: frame.width - 48, height: 42)
        toolsLabel.autoresizingMask = [.width]
        view.addSubview(toolsLabel)
        timeToolsCompletionLabel = toolsLabel
        refreshTimeToolsCompletion()
        renderGuidedSession()
    }

    private func refreshTimeToolsCompletion() {
        guard let snapshot = try? timeToolsFiles.loadSnapshot() else { return }
        let text = timeToolsOverlayText(snapshot: snapshot)
        if timeToolsCompletionLabel?.stringValue != text {
            timeToolsCompletionLabel?.stringValue = text
            timeToolsCompletionLabel?.setAccessibilityLabel(text)
        }
    }

    func tick() {
        refreshTimeToolsCompletion()
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

    private func loadMascotImage() -> NSImage? {
        guard let url = Bundle.module.url(
            forResource: BreakScreenVisuals.mascotResourceName,
            withExtension: "png"
        ), let image = NSImage(contentsOf: url) else {
            return nil
        }

        image.isTemplate = false
        return image
    }

    private func makeMascotImageView(image: NSImage, frame: NSRect) -> NSImageView {
        let imageView = NSImageView(frame: frame)
        imageView.image = image
        imageView.imageScaling = .scaleProportionallyUpOrDown
        imageView.imageAlignment = .alignCenter
        imageView.wantsLayer = true
        imageView.layer?.shadowColor = NSColor.black.cgColor
        imageView.layer?.shadowOpacity = 0.22
        imageView.layer?.shadowRadius = 12
        imageView.layer?.shadowOffset = CGSize(width: 0, height: -5)
        imageView.setAccessibilityElement(true)
        imageView.setAccessibilityRole(.image)
        imageView.setAccessibilityLabel(L10n.text("A hamster resting comfortably"))
        return imageView
    }

    private func startMascotAnimation() {
        guard let mascotImageView, let layer = mascotImageView.layer else { return }

        if NSWorkspace.shared.accessibilityDisplayShouldReduceMotion {
            layer.opacity = 1
            return
        }

        let scale = CGFloat(BreakScreenVisuals.breathingScale)
        let contracted = CATransform3DMakeScale(1 - scale, 1 - scale, 1)
        let expanded = CATransform3DMakeScale(1 + scale, 1 + scale, 1)

        let breathing = CAKeyframeAnimation(keyPath: "transform")
        breathing.values = [
            NSValue(caTransform3D: contracted),
            NSValue(caTransform3D: expanded),
            NSValue(caTransform3D: contracted),
        ]
        breathing.keyTimes = [0, 0.5, 1]
        breathing.duration = BreakScreenVisuals.breathingDuration
        breathing.timingFunctions = [
            CAMediaTimingFunction(name: .easeInEaseOut),
            CAMediaTimingFunction(name: .easeInEaseOut),
        ]
        breathing.repeatCount = .greatestFiniteMagnitude
        breathing.isRemovedOnCompletion = false

        layer.opacity = 1
        layer.add(breathing, forKey: "breakScreen.mascot.breathe")

        let fadeIn = CABasicAnimation(keyPath: "opacity")
        fadeIn.fromValue = 0
        fadeIn.toValue = 1
        fadeIn.duration = BreakScreenVisuals.entryFadeDuration
        fadeIn.timingFunction = CAMediaTimingFunction(name: .easeOut)
        layer.add(fadeIn, forKey: "breakScreen.mascot.fadeIn")
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
        progressView.setAccessibilityValue(L10n.text("{0} percent", Int((progress * 100).rounded())))

        if elapsed >= args.skipAfter && !skipButton.isEnabled {
            skipButton.isEnabled = true
            skipButton.title = L10n.text("Skip Break")
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
            guideEyebrowLabel.stringValue = L10n.text("2-minute guide")
            guideEyebrowLabel.frame = NSRect(x: 24, y: 164, width: cardWidth - 48, height: 18)

            guideTitleLabel.stringValue = L10n.text("Stand and loosen your neck and shoulders for 2 minutes")
            guideTitleLabel.frame = NSRect(x: 24, y: 110, width: cardWidth - 48, height: 48)

            guideCountdownLabel.isHidden = true

            let instruction = canStart
                ? L10n.text("Follow along slowly on this screen.")
                : L10n.text("Less than 2 minutes remain in this break.")
            guideInstructionLabel.stringValue = instruction
            guideInstructionLabel.frame = NSRect(x: 24, y: 66, width: cardWidth - 48, height: 36)

            guideStatusLabel.isHidden = true

            guideActionButton.isHidden = false
            guideActionButton.isEnabled = canStart
            guideActionButton.title = L10n.text("Start")
            guideActionButton.action = #selector(startGuidedBreak)
            guideActionButton.contentTintColor = canStart ? .white : NSColor(white: 0.5, alpha: 1.0)
            guideActionButton.frame = NSRect(x: (cardWidth - 120) / 2, y: 14, width: 120, height: 44)
            guideActionButton.setAccessibilityLabel(L10n.text("Start a 2-minute neck and shoulder stretch"))
            guideActionButton.setAccessibilityHelp(L10n.text("Starts the 2-minute guide on this screen."))

            guideCardView.setAccessibilityLabel(L10n.text("2-minute guide"))
            guideCardView.setAccessibilityValue("\(guideTitleLabel.stringValue). \(instruction)")

        case let .running(remainingSeconds):
            let instruction = guidedSession.instructionText()

            guideEyebrowLabel.isHidden = true

            guideTitleLabel.stringValue = L10n.text("Neck and shoulder stretch")
            guideTitleLabel.frame = NSRect(x: 24, y: 158, width: cardWidth - 48, height: 28)

            guideCountdownLabel.isHidden = false
            guideCountdownLabel.stringValue = formatTime(remainingSeconds)
            guideCountdownLabel.frame = NSRect(x: 24, y: 100, width: cardWidth - 48, height: 56)
            guideCountdownLabel.setAccessibilityLabel(L10n.text("Guide time remaining"))
            guideCountdownLabel.setAccessibilityValue(accessibilityDuration(remainingSeconds))

            guideInstructionLabel.stringValue = instruction
            guideInstructionLabel.frame = NSRect(x: 24, y: 56, width: cardWidth - 48, height: 42)

            guideStatusLabel.isHidden = true

            guideActionButton.isHidden = false
            guideActionButton.isEnabled = true
            guideActionButton.title = L10n.text("Cancel guide")
            guideActionButton.action = #selector(cancelGuidedBreak)
            guideActionButton.contentTintColor = .white
            guideActionButton.frame = NSRect(x: (cardWidth - 140) / 2, y: 8, width: 140, height: 44)
            guideActionButton.setAccessibilityLabel(L10n.text("Cancel guide"))
            guideActionButton.setAccessibilityHelp(L10n.text("Stops the guide and returns to the break screen."))

            guideCardView.setAccessibilityLabel(L10n.text("Neck and shoulder stretch"))
            guideCardView.setAccessibilityValue(
                L10n.text("Remaining: {0}, {1}", accessibilityDuration(remainingSeconds), instruction)
            )

        case .completed:
            guideEyebrowLabel.isHidden = true

            guideTitleLabel.stringValue = L10n.text("All done")
            guideTitleLabel.frame = NSRect(x: 24, y: 142, width: cardWidth - 48, height: 32)

            guideCountdownLabel.isHidden = true

            guideInstructionLabel.stringValue = L10n.text("Relax and enjoy the rest of your break.")
            guideInstructionLabel.frame = NSRect(x: 24, y: 86, width: cardWidth - 48, height: 48)

            guideStatusLabel.isHidden = false
            guideStatusLabel.stringValue = L10n.text("This screen closes in 3 seconds.")
            guideStatusLabel.frame = NSRect(x: 24, y: 54, width: cardWidth - 48, height: 22)

            guideActionButton.isHidden = true
            guideActionButton.isEnabled = false

            guideCardView.setAccessibilityLabel(L10n.text("All done"))
            guideCardView.setAccessibilityValue(guidedSession.instructionText())
            guideCardView.setAccessibilityHelp(L10n.text("This screen closes in 3 seconds."))
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
                .announcement: L10n.text("You completed the 2-minute stretch."),
                .priority: NSAccessibilityPriorityLevel.high.rawValue,
            ]
        )
    }

    private func accessibilityDuration(_ seconds: Int) -> String {
        let clampedSeconds = max(0, seconds)
        return L10n.text("{0} min {1} sec", clampedSeconds / 60, clampedSeconds % 60)
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
