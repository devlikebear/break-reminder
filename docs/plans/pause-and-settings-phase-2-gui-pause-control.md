# Phase 2: GUI 멈춤 컨트롤 — 작업지시서

_작성일: 2026-05-12_
_속한 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)_
_예상 소요: 3-4시간_

## 페이즈 목표

대시보드 GUI에서 마우스 클릭만으로 회의/집중/외출 멈춤을 시작하고 해제할 수 있다. 현재 상태(워킹/브레이크/일시정지+모드)가 헤더에 시각적으로 표시되며, state 파일에는 정확히 `PAUSE_REASON`이 기록된다. 백엔드 호출은 CLI Process 호출로 위임 (검증/락 활용).

## 전제 조건

- [ ] Phase 1 완료 및 사용자 승인
- [ ] `break-reminder pause --mode=...` / `resume`이 CLI에서 정상 동작 확인
- [ ] Xcode 또는 Swift 5.x 빌드 환경 준비됨 (`cd helpers && swift build`)

## 포함 기능

1. `AppState.pauseReason` 필드 + `StateParser`의 `PAUSE_REASON` 파싱
2. `DashboardViewModel`에 `pause(mode:)` / `resume()` 액션 (CLI Process 호출)
3. `StatusHeaderView`에 멈춤 컨트롤 UI 추가:
   - Working/Break 상태일 때: "⏸ Pause" 버튼 + 모드 선택 (segmented control or menu)
   - Paused 상태일 때: "▶︎ Resume" 버튼 + 현재 모드 라벨
4. 멈춤 모드별 색상/아이콘 차별화 (예: meeting=파랑, focus=주황, afk=회색)

## 이 페이즈에서 하지 않는 것

- Auto-resume duration picker → Phase 3
- 설정 탭 → Phase 5
- 멈춤 통계 표시 (오늘 회의로 멈춘 시간 등) → Out of Scope
- TUI 대시보드(`internal/dashboard`) 변경 → Out of Scope

## 작업 체크리스트

### 작업 그룹 A: HelperCore 파서 확장

- [ ] **T2.A.1** — `AppState.pauseReason` 추가
  - 파일: `helpers/Sources/HelperCore/StateParser.swift`
  - 내용:
    - `AppState` struct에 `public var pauseReason: String = ""` 추가 (`pausedAt` 다음 줄)
    - `parseState(from:)` switch에 `case "PAUSE_REASON": s.pauseReason = val` 추가
    - `serializeState(_:)`에 `"PAUSE_REASON=\(s.pauseReason)"` 추가 (PAUSED_AT 다음 줄)
  - 참조: 기존 다른 필드들의 처리 방식 그대로
  - 검증: `swift build` 통과

- [ ] **T2.A.2** — HelperCore 단위 테스트 (있다면)
  - 파일: `helpers/Tests/HelperCoreTests/StateParserTests.swift` (존재 여부 확인)
  - 내용:
    - 파일 없으면 이 작업 스킵 (기존에 테스트 없음)
    - 있으면 PAUSE_REASON round-trip 테스트 추가
  - 검증: `swift test` 통과 (테스트 있을 때만)

### 작업 그룹 B: ViewModel 액션

- [ ] **T2.B.1** — `DashboardViewModel.pause(mode:)` 추가
  - 파일: `helpers/Sources/DashboardApp/DashboardViewModel.swift`
  - 내용:
    - 새 메서드:
      ```swift
      func pause(mode: String) {
          runCLI(args: ["pause", "--mode=\(mode)"])
          refresh()
      }

      func resume() {
          runCLI(args: ["resume"])
          refresh()
      }
      ```
    - 보조 유틸 (private):
      ```swift
      private func runCLI(args: [String]) {
          guard let cli = findHelper("break-reminder") else { return }
          let p = Process()
          p.launchPath = cli
          p.arguments = args
          p.standardOutput = FileHandle.nullDevice
          p.standardError = FileHandle.nullDevice
          do {
              try p.run()
              p.waitUntilExit()
          } catch {
              // log to console; non-fatal
          }
      }
      ```
  - 참조: 기존 `runInsightsRefresh()`가 동일한 패턴
  - 검증: 컴파일 통과 (실제 동작은 T2.C.2 이후 수동 확인)

- [ ] **T2.B.2** — `pauseModeLabel` / `pauseModeColor` 계산 속성
  - 파일: `helpers/Sources/DashboardApp/DashboardViewModel.swift`
  - 내용:
    - `pauseModeLabel: String` — `state.pauseReason`을 한국어로 변환 ("회의", "집중", "외출", 빈 값이면 "")
    - `pauseModeAccent: Color` — 모드별 색상 (예: meeting → `.blue`, focus → `.orange`, afk → `.gray`, default → `.secondary`)
    - `statusText` 변경: 기존 `"PAUSED (\(isWork ? "WORK" : "BREAK"))"` → `"PAUSED · \(pauseModeLabel)"` (모드가 비어있으면 `"PAUSED"`)
  - 검증: 컴파일 통과

### 작업 그룹 C: StatusHeaderView UI

- [ ] **T2.C.1** — 멈춤/재개 버튼 + 모드 선택 UI
  - 파일: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
  - 내용:
    - 기존 헤더 레이아웃 안에 새 컴포넌트 영역 추가 (`HStack` 하단 또는 우측)
    - 분기:
      - `vm.isPaused` 가 `true`: "▶ Resume" 버튼 (`.borderedProminent` 스타일), 좌측에 현재 모드 라벨 (예: "Paused · 회의"). 클릭 시 `vm.resume()`.
      - `vm.isPaused` 가 `false`: "⏸ Pause" Menu(`SwiftUI.Menu`) — 클릭 시 3가지 모드 선택지 드롭다운:
        - "회의 (Meeting)" → `vm.pause(mode: "meeting")`
        - "집중 (Focus)" → `vm.pause(mode: "focus")`
        - "외출 (AFK)" → `vm.pause(mode: "afk")`
    - 각 모드 메뉴 항목에 짧은 설명 가능하면 SF Symbol + 한 줄 부연 표시
  - 참조: SwiftUI Menu 사용법은 표준. 기존 forceBreak 버튼이 어디 배치되어 있는지 확인하고 인접 배치.
  - 검증: `cd helpers && swift build` 통과. 실행 시 빌드 에러 없이 헤더에 새 컨트롤 표시.

- [ ] **T2.C.2** — 모드 색상 차별화
  - 파일: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
  - 내용:
    - paused 상태일 때 라벨/배지 색상에 `vm.pauseModeAccent` 적용
    - 다크/라이트 테마 양쪽 가독성 확인 (ThemeManager 활용)
  - 검증: 수동 — 다크 모드와 라이트 모드에서 멈춤 시 모드별 색이 보이는지 확인

### 작업 그룹 D: 통합 확인

- [ ] **T2.D.1** — 전체 빌드 + 데몬 함께 실행 시나리오
  - 파일: 없음 (수동 검증)
  - 내용:
    - `make build` 또는 `go build ./cmd/break-reminder` 후 바이너리가 GUI에서 찾는 경로(`~/.local/bin/break-reminder` 또는 Bundle 내부)에 있는지 확인
    - Dashboard 빌드: `cd helpers && swift build`
    - 실행: `swift run DashboardApp` (또는 빌드 결과 직접 실행)
  - 검증: GUI에서 멈춤 메뉴 → 모드 선택 → 1초 내에 상태 변화. `~/.break-reminder-state` 파일에 `PAUSE_REASON=focus` 등 라인 존재. Resume 클릭 후 `PAUSE_REASON=`로 초기화.

---

## ✅ Phase 2 Checkpoint

**구현 확인:**
- [ ] 모든 작업 체크박스 완료
- [ ] `AppState.pauseReason` 필드 + 파서/직렬화 연동
- [ ] GUI 헤더에 Pause 메뉴 + Resume 버튼 렌더링됨
- [ ] 3가지 모드 모두 클릭 가능

**자동 검증:**
- [ ] Go 빌드/테스트: `go test ./...` 통과 (회귀 없음)
- [ ] Swift 빌드: `cd helpers && swift build` 통과
- [ ] Swift 테스트(있으면): `cd helpers && swift test` 통과

**수동 확인:**
- [ ] GUI에서 "회의 (Meeting)" 선택 → 헤더가 "PAUSED · 회의"로 바뀌고, state 파일에 `PAUSE_REASON=meeting` 기록
- [ ] GUI Resume 클릭 → 헤더가 "WORKING"으로 복귀, state 파일에서 `PAUSE_REASON=` 초기화
- [ ] focus / afk 모드 동일하게 작동, 라벨과 색상이 모드별로 다름
- [ ] CLI `break-reminder pause --mode=focus`로 멈춘 상태에서 GUI를 켜면 헤더에 "PAUSED · 집중" 정확히 반영

**완료 처리:**
1. 사용자에게 완료 보고 (구현/검증/수동 결과)
2. 사용자 승인 후 Phase 3로 이동.
3. 실패 시: 원인 분석 → 수정 → 재검증.

---

## 참고 자료

- 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)
- 기존 GUI CLI 호출 패턴: `DashboardViewModel.runInsightsRefresh()` (line ~125)
- 기존 GUI state write 패턴: `DashboardViewModel.forceBreak()` (line ~159) — **이번엔 직접 write 대신 CLI 호출**
- 기존 헤더 UI: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
- 테마 시스템: `ThemeManager.swift`

## 메모 / 주의

- **왜 state 직접 write 대신 CLI 호출?**: `internal/state.Update`는 파일락(`syscall.Flock`)으로 데몬과의 race condition을 방지한다. Swift에서 직접 write하면 락을 건너뛰어 데몬 tick과 충돌 가능. CLI 호출은 `Update`를 거치므로 안전.
- **성능**: CLI Process 호출은 ~50-100ms. 멈춤/재개는 자주 일어나지 않으므로 허용 가능.
- **에러 처리**: 현재 단계에선 CLI 실패를 silent로 처리하고 `refresh()`로 실제 상태를 다시 읽어 화면 갱신. UX는 "버튼 눌렀는데 상태 안 바뀜"으로 사용자가 알아차림. 정교한 에러 UI는 Phase 5에서 Settings 저장 실패 다룰 때 일관되게.
- **헤더 공간**: 현재 헤더가 이미 조밀할 수 있음. 멈춤 버튼은 우측 정렬, 모드 선택은 Menu(드롭다운)로 펼치는 게 공간 효율적.

---
_다음 페이즈: Phase 3 — Auto-resume 타이머 → [`pause-and-settings-phase-3-auto-resume.md`](./pause-and-settings-phase-3-auto-resume.md)_
