import Foundation

public enum AppLanguage: String {
    case english = "en", korean = "ko"
    public static func resolve(_ tag: String) -> Self {
        tag.lowercased().components(separatedBy: CharacterSet(charactersIn: "-_.@")).first == "ko" ? .korean : .english
    }
    public static var current: Self {
        if let override = ProcessInfo.processInfo.environment["BREAK_REMINDER_LANGUAGE"], !override.isEmpty {
            return resolve(override)
        }
        return resolve(Locale.preferredLanguages.first ?? "en")
    }
}

public enum L10n {
    public static func text(_ key: String, _ arguments: Any..., language: AppLanguage = .current) -> String {
        let template = language == .korean ? (korean[key] ?? key) : key
        let pattern = try! NSRegularExpression(pattern: #"\{(\d+)\}"#)
        let result = NSMutableString(string: template)
        // Replace template ranges in reverse order; user text is never interpreted.
        for match in pattern.matches(in: template, range: NSRange(template.startIndex..., in: template)).reversed() {
            let number = (template as NSString).substring(with: match.range(at: 1))
            if let index = Int(number), index < arguments.count {
                result.replaceCharacters(in: match.range, with: String(describing: arguments[index]))
            }
        }
        return result as String
    }
}
