import AppKit
import HelperCore

/// Loads the traced full-color SVG hamster frames and caches the resulting vector images.
enum HamsterMenuBarIcon {
    static let size = NSSize(width: 24, height: 18)

    private static let runningImages = HamsterVectorArtwork.running.map { makeImage(svg: $0) }
    private static let restingImages = HamsterVectorArtwork.resting.map { makeImage(svg: $0) }
    private static let sleepingImage = makeImage(svg: HamsterVectorArtwork.sleeping)

    static func image(for frame: MenuBarAnimationFrame) -> NSImage {
        func wrapped(_ value: Int, count: Int) -> Int {
            ((value % count) + count) % count
        }

        switch frame.kind {
        case .running:
            return runningImages[wrapped(frame.frameIndex, count: runningImages.count)]
        case .resting:
            return restingImages[wrapped(frame.frameIndex, count: restingImages.count)]
        case .sleeping:
            return sleepingImage
        }
    }

    private static func makeImage(svg: String) -> NSImage {
        guard let image = NSImage(data: Data(svg.utf8)) else {
            NSLog("Break Reminder: embedded hamster SVG could not be decoded; using fallback icon")
            let fallback = NSImage(
                systemSymbolName: "pawprint.fill",
                accessibilityDescription: "Break Reminder"
            ) ?? NSImage(size: size)
            fallback.size = size
            fallback.isTemplate = true
            return fallback
        }
        image.size = size
        image.isTemplate = false
        return image
    }
}
