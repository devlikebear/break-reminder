# About(정보) 메뉴 — Design Spec

> Date: 2026-09-01
> Status: Approved
> Target: MenuBarApp, DashboardApp (Swift), TUI dashboard + status (Go)

## Overview

설치된 break-reminder 버전을 UI에서 확인할 수 없다. `break-reminder version`을 터미널에서 직접 치는 것 외에는 경로가 없어, 사용자가 "지금 무슨 버전이 깔려 있는지 모르겠다"는 상태에 놓인다.

메뉴바 앱과 GUI 대시보드에 About(정보) 항목을 추가해 버전과 저장소 링크를 노출한다. 부수적으로 TUI 대시보드와 `status` 명령에도 버전을 표시해 확인 경로를 넓힌다.

## Problem: Swift 헬퍼는 버전을 모른다

버전은 Go 바이너리에만 박혀 있다.

- `VERSION` 파일 → Makefile이 `-X main.version=$(VERSION)` ldflags로 주입 ([Makefile:8-9](../../../Makefile))
- `cmd/break-reminder/main.go`의 `version` 변수가 유일한 보유자, `break-reminder version`으로 출력
- Swift 헬퍼(MenuBarApp / DashboardApp)는 버전 정보를 전혀 갖고 있지 않다

### 검토한 대안

| 방식 | 장점 | 채택하지 않은 이유 |
|------|------|------------------|
| **A. `break-reminder version` 서브프로세스 호출** | 빌드 변경 0, 설치된 CLI와 항상 일치, 기존 패턴 재사용 | — (채택) |
| B. 빌드 시 `Version.swift` 생성 | 서브프로세스 없음 | 생성 파일 gitignore 관리 필요. CI가 `cd helpers && swift test`로 Makefile을 우회해 깨진다. 무엇보다 헬퍼 바이너리만 옛 버전으로 남으면 **틀린 버전을 자신 있게 표시**해 원래 문제를 악화시킨다 |
| C. 설치 시 버전 파일 기록 | 파싱이 단순 | 새 파일의 생명주기(삭제·누락·Homebrew 경로)를 전부 책임져야 해 관리 표면이 넓다 |

**A 채택.** 단일 진실 공급원이 "설치된 `break-reminder` 바이너리" 하나로 유지된다. 두 헬퍼 모두 이미 `findHelper()`로 형제 바이너리를 찾아 `Process`로 실행하는 패턴을 갖고 있다 (MenuBarApp의 `toggleSession`, DashboardApp의 `config set`).

## Architecture

### 공유 레이어 — `helpers/Sources/HelperCore/VersionInfo.swift` (신규)

HelperCore는 순수 파싱/포맷만, 앱은 IO를 담당하는 기존 분리를 따른다.

```swift
public enum AboutInfo {
    public static let appName = "Break Reminder"
    public static let repositoryURL = "https://github.com/devlikebear/break-reminder"
    public static let unknownVersion = "unknown"
}

/// "break-reminder 0.13.0" → "0.13.0"
public func parseVersionOutput(_ raw: String) -> String
```

파싱 규칙:

1. 첫 번째 비어 있지 않은 줄을 취한다
2. 공백 기준으로 나눈 뒤 마지막 토큰을 반환한다
3. 입력이 비었거나 토큰이 없으면 `AboutInfo.unknownVersion`

`break-reminder dev`(ldflags 없이 빌드한 경우)도 그대로 `dev`로 통과한다.

### IO 레이어

각 앱이 `findHelper("break-reminder")`로 경로를 찾고 `version` 인자로 실행한 뒤 `parseVersionOutput`에 넘긴다. **앱 수명 동안 1회만 호출하고 캐싱한다.**

CLI를 찾지 못하면 `unknown`을 표시한다. 이는 실제로 설치가 깨진 상태이므로 정직한 신호다.

## Components

### 1. 메뉴바 About

`helpers/Sources/MenuBarApp/main.swift`

- `Open Config` 항목 아래, Quit 앞 구분선 위에 `About Break Reminder` 항목 추가
- 액션: NSAlert
  - `messageText`: `Break Reminder`
  - `informativeText`: `Version 0.13.0`
  - 버튼: `GitHub` / `Close`
- `GitHub` 선택 시 `NSWorkspace.shared.open(AboutInfo.repositoryURL)`
- 메뉴바 앱은 `.accessory` 정책이라 알림이 뒤로 숨을 수 있다. `runModal()` 직전에 `NSApp.activate(ignoringOtherApps: true)`를 호출한다
- 기존 메뉴 항목이 모두 영문이므로 문구도 영문으로 맞춘다
- 버전은 컨트롤러에 1회 캐싱

### 2. GUI 대시보드 정보 섹션

`helpers/Sources/DashboardApp/SettingsTabView.swift`, `DashboardViewModel.swift`

`buttonRow`(저장/취소) 다음에 `Divider()` + 기존 `sectionCard(title: "정보")`를 붙인다. 저장 버튼은 폼에 속하므로 폼 바로 아래 유지하고, 정보는 그 밑 별개 카드로 둔다.

```
정보
┌──────────────────────────────┐
│ 버전              0.13.0      │
│ [ GitHub 저장소 열기 ]         │
└──────────────────────────────┘
```

- `DashboardViewModel`에 `@Published private(set) var appVersion: String` 추가, 초기값은 `AboutInfo.unknownVersion`
- `start()`에서 1회 채운다. ViewModel은 이미 `config set`으로 CLI를 호출하므로 자연스러운 자리다
- 대시보드는 한국어 UI이므로 라벨도 한국어

### 3. TUI 대시보드

`internal/dashboard/dashboard.go`, `cmd/break-reminder/dashboard.go`

TUI에는 별도 푸터가 없고 조작 힌트가 타이틀 줄에 함께 있다 ([dashboard.go:199](../../../internal/dashboard/dashboard.go)). 버전은 그 줄에 붙인다.

```
🐹 Break Reminder Dashboard v0.13.0 (q:quit r:reset b:break s:start/stop)
```

`dashboard.New(cfg)` → `dashboard.New(cfg, version)`로 시그니처가 바뀐다. 호출부는 `cmd/break-reminder/dashboard.go`와 기존 테스트다.

### 4. status 명령

`cmd/break-reminder/status.go`

구분선 바로 아래 `Version: 0.13.0` 한 줄을 추가한다. 기존 출력의 `키: 값` 스타일을 따른다.

```
🐹 Break Reminder Status
========================
Version: 0.13.0
System: Installed & Running
Menu Bar: Installed & Running
...
```

`status.go`는 `main` 패키지이므로 `version` 변수를 그대로 읽는다. TUI와 달리 시그니처 변경이 필요 없다.

## Testing

| 대상 | 검증 |
|------|------|
| `HelperCoreTests/VersionInfoTests.swift` (신규) | `parseVersionOutput`: 정상값, `dev`, 빈 문자열, 개행 포함, 다중 줄, 공백만 있는 입력 |
| `cmd/break-reminder/status_test.go` | 출력에 버전 줄이 포함되는지 |
| `internal/dashboard/dashboard_test.go` | 타이틀 줄에 전달된 버전이 포함되는지 |
| 수동 | `make install` 후 메뉴바 About 항목, 설정 탭 하단 정보 섹션 |

프로세스 실행 자체는 테스트하지 않는다. IO 경계이고 기존 `findHelper` 호출부들도 같은 방침이다.

## Out of Scope

- **업데이트 확인 버튼** — `break-reminder update`와 autoupdate 패키지가 이미 있지만, 비동기 진행 상태·실패 처리 UI가 붙으면 범위가 크게 늘어난다
- **진단 정보(설치 경로, launchd 상태)** — `doctor` 명령과 역할이 겹친다
- **빌드 시스템 변경** — Makefile과 릴리스 워크플로는 손대지 않는다

## Non-Goals / 주의

- 헬퍼가 표시하는 버전은 **설치된 CLI의 버전**이지 헬퍼 바이너리 자체의 버전이 아니다. 이는 의도된 설계다 (대안 B 참조)
