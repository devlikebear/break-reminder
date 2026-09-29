# Phase 1 — 포모도로를 메뉴바와 대시보드에서 사용

로드맵: [time-tools-roadmap.md](time-tools-roadmap.md) · 공통 계약: [time-tools-design.md](time-tools-design.md)

예상 12–20시간. 목표는 기존 포모도로를 GUI에서 시작·확인하는 완결된 흐름이다. 일반 타이머/worker/새 저장소는 아직 만들지 않는다.

## 시작 동작의 정의

GUI `포모도로 시작`은 정지 상태에서 **`pomodoro on`(모드 선택·구간 초기화) 성공 후 `start`(업무 세션 시작)**까지 수행하는 복합 동작이다. 기존 CLI `pomodoro on` 자체의 의미는 바꾸지 않는다. 이미 pomodoro로 실행 중이면 중복 시작하지 않는다. 설정에서 모드만 선택하는 동작과 시작 버튼을 구분한다. 부분 성공·재시도 정책은 공통 설계 2절을 따른다.

## 전제

- [ ] 현재 git 상태와 테스트 baseline 기록. 기존 미추적 파일 보존.
- [ ] 기존 `pomodoro on/off/status`, `start`, `pause`, `resume`의 의미 확인.
- [ ] 현재 포모도로가 활동 기반이라는 안내를 유지하고 고정 25분 타이머로 설명하지 않기.

## 작업 목록

### P1.1 CLI 실패를 UI에서 판별할 수 있게 만들기 — 2–3시간

- [ ] `cmd/break-reminder/session_cmd_test.go` 또는 신규 `pomodoro_test.go`에 상태 저장 실패 RED 작성. config 저장 성공/state reset 실패 시 명령이 성공으로 종료하면 실패해야 한다.
- [ ] `pomodoro.go:resetCurrentCycle`을 error 반환으로 바꾸고 on/off에서 전파한다. config가 이미 저장됐다는 부분 성공 정보도 오류 문구에 포함한다.
- [ ] 기존 시간 옵션 검증 테스트 추가: 명시한 0/음수는 조용히 기본값으로 무시하지 말고 오류. 생략과 명시 0은 `Flags().Changed`로 구분. 실제 최대값은 기존 config validator와 동일하게 유지.
- [ ] 회귀: on/off의 일일 통계 보존, 기존 사용자 지정 시간 유지, work cycle reset, pause/break에서의 기존 CLI 정책은 의도 없이 바꾸지 않는다.
- [ ] 검증: `go test ./cmd/break-reminder -run 'TestPomodoro'`, `go test ./internal/timer ./internal/state`.

### P1.2 공유 비동기 명령 어댑터 — 2–3시간

- [ ] `helpers/Tests/HelperCoreTests/CLICommandClientTests.swift` RED: 성공, nonzero stderr, 바이너리 없음, timeout, stdout/stderr 동시 대용량 출력, 중복 클릭 억제.
- [ ] 신규 `HelperCore/CLICommandClient.swift`: `run(arguments: [String]) async -> CommandResult` 및 주입 가능한 실행 protocol. `CommandResult`는 exitCode/stdout/stderr/timeout을 구분한다. 제어 명령 timeout은 10초.
- [ ] 경로 탐색은 기존 sibling/Homebrew/사용자 설치 경로를 보존한다. 공통화할 경우 Dashboard와 MenuBar 양쪽 테스트 추가. shell 실행 금지.
- [ ] UI 실행 흐름을 HelperCore의 테스트 가능한 controller/policy로 분리한다. executable target 전체를 테스트 dependency로 넣지 않는다.
- [ ] `DashboardViewModel`과 `MenuBarApp`의 신규 조작만 우선 연결. 기존 AI 실행기나 모든 legacy CLI 호출을 한꺼번에 재작성하지 않는다.
- [ ] 검증: `swift test --package-path helpers --filter CLICommandClientTests`.

### P1.3 포모도로 조작·표현 정책 — 2–3시간

- [ ] 신규 `HelperCore/PomodoroPresentation.swift`와 테스트: classic/활성/정지/휴식/paused별 가능한 동작, 1/4~4/4 회차, 긴 휴식, 오늘 누계.
- [ ] 순차 command 정책 테스트: stopped에서 on→start, 활성 pomodoro에서 no-op, on 실패 시 start 금지, on 성공/start 실패 시 부분 성공 안내, retry는 start만 호출.
- [ ] 새 `PomodoroControlsView.swift`와 기존 `TimerTabView.swift`에 모드 선택·상태·시작·설정 진입점 추가. settings에서 작업/짧은 휴식/긴 휴식/간격 값을 편집하되 running/paused/break 중 편집 비활성.
- [ ] 메뉴바에 빠른 포모도로 시작과 classic 전환 제공. 휴식/paused에서는 disabled 설명, classic 실행 중 전환은 현재 구간 초기화 확인.
- [ ] 모든 성공/실패 후 기존 파일을 다시 읽고 실제 상태 반영. main thread blocking이나 성공 전 checkmark 변경 금지.

### P1.4 텍스트 입력·화면 검증 — 2–3시간

- [ ] 신규 `HelperCore/ShortcutPolicy.swift` 테스트: 텍스트 편집/수정키/일반 화면 상황 구분.
- [ ] `DashboardAppMain.swift:installKeyMonitor`에서 field editor/텍스트 입력 중 Q/R/B를 통과시킨다. monitor 중복 등록·해제 생명주기도 확인한다.
- [ ] 배경 클릭 focus 강제가 입력칸을 방해하지 않는지 조정. 설정 입력에 `qrb`가 포함돼도 quit/reset/force break가 실행되지 않아야 한다.
- [ ] 360×600에서 ScrollView와 키보드 이동 확인. 기존 헤더·통계·설정 탭을 숨기지 않는다.

## Checkpoint

- [ ] targeted RED/GREEN 증거와 `make test`, `make build` 결과 기록.
- [ ] 격리된 프로필에서 메뉴바 시작 → 대시보드 같은 회차 → 일시정지/재개 → classic 복귀를 실행.
- [ ] 기본 60초 tick을 주입한 Go 테스트로 25/5·4회·긴 휴식 검증. 실시간 2시간 기다리기로 자동 검증을 대체하지 않는다.
- [ ] 실제 짧은 사용자 설정으로 휴식 진입을 확인하고 활동 유휴 영향이 UI 설명과 일치하는지 기록.
- [ ] CLI 실패 시 오류가 보이고 UI가 실제 저장 상태를 유지한다.
- [ ] README/CHANGELOG Unreleased에 활동 기반 포모도로 조작 설명 추가.

실패는 원인과 영향을 기록하고 해당 작업을 수정한다. 통과 후 결과를 보고하고 승인된 구현 범위에 따라 [Phase 2](time-tools-phase-2-countdown.md)로 진행한다.
