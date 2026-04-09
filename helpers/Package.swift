// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "BreakReminderHelpers",
    platforms: [.macOS(.v12)],
    targets: [
        .target(
            name: "HelperCore"
        ),
        .executableTarget(
            name: "BreakScreenApp",
            dependencies: ["HelperCore"]
        ),
        .executableTarget(
            name: "DashboardApp",
            dependencies: ["HelperCore"]
        ),
        .executableTarget(
            name: "MenuBarApp",
            dependencies: ["HelperCore"]
        ),
        .testTarget(
            name: "HelperCoreTests",
            dependencies: ["HelperCore"]
        ),
    ]
)
