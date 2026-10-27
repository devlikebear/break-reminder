import AppKit
import HelperCore
import ServiceManagement
import SwiftUI

/// Status bar item: left click opens the SwiftUI popover, right click the menu.
@MainActor
final class StatusItemController: NSObject, NSMenuDelegate {
    private let model: AppModel
    private let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let popover = NSPopover()
    private let menu = NSMenu()
    private var refreshTimer: Timer?
    private var animationTick = 0
    private var loginItem: NSMenuItem!

    init(model: AppModel, calendar: CalendarCollector) {
        self.model = model
        super.init()
        popover.behavior = .transient
        popover.contentViewController = NSHostingController(rootView: PopoverRootView(model: model, calendar: calendar))

        if let button = statusItem.button {
            button.imagePosition = .imageLeading
            button.imageScaling = .scaleProportionallyDown
            button.target = self
            button.action = #selector(clicked(_:))
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }
        buildMenu()
        refresh()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 0.25, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.tick() }
        }
    }

    private func tick() {
        animationTick = (animationTick + 1) % 12_000
        if animationTick % 4 == 0 { refresh() } else { updateIcon() }
    }

    private func refresh() {
        guard let status = model.status else {
            statusItem.button?.title = model.engineRunning ? "" : " !"
            updateIcon()
            return
        }
        let now = Int64(Date().timeIntervalSince1970)
        var title = popoverPresentation(status: status, now: now).menuTitle
        if let tools = status.timeTools, let toolsTitle = timeToolsMenuTitle(snapshot: tools, nowMS: now * 1000) {
            title = toolsTitle
        }
        statusItem.button?.title = title
        updateIcon()
    }

    private func updateIcon() {
        let state = model.status?.appState ?? AppState()
        let config = model.status?.appConfig ?? AppConfig()
        statusItem.button?.image = HamsterMenuBarIcon.image(for: menuBarAnimation(state: state, config: config, tick: animationTick))
    }

    @objc private func clicked(_ sender: NSStatusBarButton) {
        if NSApp.currentEvent?.type == .rightMouseUp {
            statusItem.menu = menu
            statusItem.button?.performClick(nil)
            return
        }
        if popover.isShown {
            popover.performClose(sender)
        } else {
            Task { await model.refresh() }
            popover.show(relativeTo: sender.bounds, of: sender, preferredEdge: .minY)
            popover.contentViewController?.view.window?.makeKey()
        }
    }

    func menuDidClose(_ menu: NSMenu) {
        // Detach so the next left click reaches `clicked` instead of the menu.
        statusItem.menu = nil
    }

    func menuWillOpen(_ menu: NSMenu) {
        loginItem.state = SMAppService.mainApp.status == .enabled ? .on : .off
    }

    private func buildMenu() {
        menu.delegate = self
        menu.autoenablesItems = false
        func item(_ title: String, _ action: Selector, key: String = "") -> NSMenuItem {
            let i = NSMenuItem(title: title, action: action, keyEquivalent: key)
            i.target = self
            menu.addItem(i)
            return i
        }
        item(L10n.text("Start Work Session") + " / " + L10n.text("Stop Work Session"), #selector(toggleSession), key: "s")
        item(L10n.text("Reset Timer"), #selector(resetTimer), key: "r")
        menu.addItem(.separator())
        item(L10n.text("Open Config"), #selector(openConfig), key: ",")
        loginItem = item(L10n.text("Launch at login"), #selector(toggleLoginItem))
        item(L10n.text("About {0}", AboutInfo.appName), #selector(showAbout))
        menu.addItem(.separator())
        item(L10n.text("Quit"), #selector(quit), key: "q")
    }

    @objc private func toggleSession() {
        model.perform(model.status?.state.sessionState == "active" ? "session.stop" : "session.start")
    }

    @objc private func resetTimer() { model.perform("timer.reset") }

    @objc private func openConfig() {
        let env = ProcessInfo.processInfo.environment
        let path: URL
        if let home = env["BREAK_REMINDER_HOME"] {
            path = URL(fileURLWithPath: home).appendingPathComponent("config.yaml")
        } else {
            path = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".config/break-reminder/config.yaml")
        }
        NSWorkspace.shared.open(path)
    }

    @objc private func toggleLoginItem() {
        do {
            if SMAppService.mainApp.status == .enabled {
                try SMAppService.mainApp.unregister()
            } else {
                try SMAppService.mainApp.register()
            }
        } catch {
            model.lastError = error.localizedDescription
        }
    }

    @objc private func showAbout() {
        NSApp.activate(ignoringOtherApps: true)
        let version = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? L10n.text("unknown")
        let alert = NSAlert()
        alert.messageText = AboutInfo.appName
        alert.informativeText = L10n.text("Version {0}", version)
        alert.addButton(withTitle: L10n.text("Close"))
        alert.runModal()
    }

    @objc private func quit() { NSApp.terminate(nil) }
}
