import Foundation

/// Static facts shown in the About/정보 surfaces of the menu bar app and the dashboard.
public enum AboutInfo {
    public static let appName = "Break Reminder"
    public static let repositoryURL = "https://github.com/devlikebear/break-reminder"

    /// Shown when the break-reminder CLI cannot be found or does not answer.
    public static let unknownVersion = "unknown"
}

/// Extracts the version from `break-reminder version` output.
///
/// The CLI prints `break-reminder 0.13.0`, so the version is the last
/// whitespace-separated token of the first non-empty line. Anything the CLI
/// puts on later lines is ignored. Returns `AboutInfo.unknownVersion` when
/// there is nothing to parse.
///
/// The helpers have no version of their own — they ask the installed CLI, which
/// carries the version injected at build time. That keeps a single source of
/// truth and means a stale helper binary can never report a wrong version.
public func parseVersionOutput(_ raw: String) -> String {
    for line in raw.components(separatedBy: .newlines) {
        let fields = line.split(whereSeparator: { $0.isWhitespace })
        guard let version = fields.last else { continue }
        return String(version)
    }
    return AboutInfo.unknownVersion
}
