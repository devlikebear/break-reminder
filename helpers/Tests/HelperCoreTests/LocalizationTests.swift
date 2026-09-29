import XCTest
@testable import HelperCore
final class LocalizationTests: XCTestCase {
    func testLanguageResolutionAndFallback() {
        XCTAssertEqual(AppLanguage.resolve("ko-KR"), .korean)
        XCTAssertEqual(AppLanguage.resolve("ko_KR.UTF-8"), .korean)
        XCTAssertEqual(AppLanguage.resolve("en-US"), .english)
        XCTAssertEqual(AppLanguage.resolve("ja-JP"), .english)
        XCTAssertEqual(AppLanguage.resolve(""), .english)
    }
    func testArgumentsAreNotTranslatedOrReinterpolated() {
        XCTAssertEqual(L10n.text("Timer complete: {0}", "Tea {1}", language: .korean), "타이머 완료: Tea {1}")
        XCTAssertEqual(L10n.text("Timer complete: {0}", "차", language: .english), "Timer complete: 차")
    }
    func testDefaultNameIsDisplayOnlyAndCustomNameIsPreserved() {
        let timer = CountdownSnapshot(id: "a", label: "", durationMS: 1000, phase: "running", deadline: 2000, remainingMS: nil, createdAt: 1000, completedAt: nil)
        XCTAssertEqual(timer.label, "")
        XCTAssertEqual(timer.displayLabel, L10n.text("Timer"))
        let recent = RecentCountdown(label: "차 {0}", durationMS: 1000)
        XCTAssertEqual(recent.displayLabel, "차 {0}")
    }
    func testCommandInheritsUILanguage() async {
        let result = await CLICommandClient(executablePath: "/usr/bin/printenv").run(arguments: ["BREAK_REMINDER_LANGUAGE"])
        XCTAssertTrue(result.succeeded)
        XCTAssertEqual(result.stdout.trimmingCharacters(in: .whitespacesAndNewlines), AppLanguage.current.rawValue)
    }
    func testCatalogHasMatchingPlaceholders() {
        for (english, korean) in L10n.korean {
            let pattern = try! NSRegularExpression(pattern: #"\{\d+\}"#)
            func fields(_ text: String) -> [String] {
                pattern.matches(in: text, range: NSRange(text.startIndex..., in: text)).map { (text as NSString).substring(with: $0.range) }.sorted()
            }
            XCTAssertEqual(fields(english), fields(korean), english)
        }
    }
}
