import Foundation

/// Shared visual contract for the fullscreen break screen.
///
/// The animation is deliberately slow and subtle so the break screen feels
/// alive without competing with the countdown or adding unnecessary wakeups.
public enum BreakScreenVisuals {
    public static let mascotResourceName = "BreakMascot"
    public static let breathingDuration: TimeInterval = 5.0
    public static let breathingScale = 0.025
    public static let entryFadeDuration: TimeInterval = 0.35
}
