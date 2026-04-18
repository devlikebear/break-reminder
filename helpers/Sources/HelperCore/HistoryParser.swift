import Foundation

public struct HistoryEntry: Equatable {
    public let date: String
    public let workMin: Int
    public let breakMin: Int
    public let sessions: Int
    public let activities: Int
    public let hourlyWork: [Int]  // Always 24 elements

    public init(date: String, workMin: Int, breakMin: Int, sessions: Int, activities: Int, hourlyWork: [Int]) {
        self.date = date
        self.workMin = workMin
        self.breakMin = breakMin
        self.sessions = sessions
        self.activities = activities
        self.hourlyWork = hourlyWork
    }
}

public func parseHistory(from json: String) -> [HistoryEntry] {
    guard let data = json.data(using: .utf8),
          let array = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]] else {
        return []
    }

    return array.map { dict in
        let hourly = dict["hourly_work"] as? [Int] ?? Array(repeating: 0, count: 24)
        let normalized = hourly.count == 24 ? hourly : Array(repeating: 0, count: 24)
        return HistoryEntry(
            date: dict["date"] as? String ?? "",
            workMin: dict["work_min"] as? Int ?? 0,
            breakMin: dict["break_min"] as? Int ?? 0,
            sessions: dict["sessions"] as? Int ?? 0,
            activities: dict["activities"] as? Int ?? 0,
            hourlyWork: normalized
        )
    }
}

public func loadHistoryFromDisk() -> [HistoryEntry] {
    let home = FileManager.default.homeDirectoryForCurrentUser
    let path = home.appendingPathComponent(".break-reminder-history.json")
    guard let content = try? String(contentsOf: path, encoding: .utf8) else { return [] }
    return parseHistory(from: content)
}
