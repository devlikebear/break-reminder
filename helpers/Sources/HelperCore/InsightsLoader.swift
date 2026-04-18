import Foundation

public struct InsightPattern: Equatable {
    public let type: String       // "warning", "positive", "info"
    public let title: String
    public let description: String
    public let suggestion: String

    public init(type: String, title: String, description: String, suggestion: String) {
        self.type = type
        self.title = title
        self.description = description
        self.suggestion = suggestion
    }
}

public struct InsightsReport: Equatable {
    public let generatedAt: String
    public let dailyReport: String
    public let patterns: [InsightPattern]

    public init(generatedAt: String, dailyReport: String, patterns: [InsightPattern]) {
        self.generatedAt = generatedAt
        self.dailyReport = dailyReport
        self.patterns = patterns
    }
}

public func parseInsights(from json: String) -> InsightsReport? {
    guard let data = json.data(using: .utf8),
          let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
        return nil
    }
    let patternsRaw = dict["patterns"] as? [[String: Any]] ?? []
    let patterns = patternsRaw.map { p in
        InsightPattern(
            type: p["type"] as? String ?? "info",
            title: p["title"] as? String ?? "",
            description: p["description"] as? String ?? "",
            suggestion: p["suggestion"] as? String ?? ""
        )
    }
    return InsightsReport(
        generatedAt: dict["generated_at"] as? String ?? "",
        dailyReport: dict["daily_report"] as? String ?? "",
        patterns: patterns
    )
}

public func loadInsightsFromDisk() -> InsightsReport? {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-insights.json")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return nil }
    return parseInsights(from: content)
}
