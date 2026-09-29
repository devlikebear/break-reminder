# 시간 도구 확장 — 코드베이스 분석

작성일: 2026-09-28 · HEAD `8e8cb795f6570adc2bc602d4124264f9e64baee8`.

분석 범위: 포모도로, 상태 파일, launchd, 알림, 메뉴바, 대시보드, 관련 테스트·빌드. 아래 내용은 현재 소스 읽기 결과이며 앱 실행/테스트 통과를 확인한 보고서가 아니다.

## 환경과 구조

- `go.mod`: Go 1.24.4, Cobra, zerolog, YAML, Bubble Tea.
- `helpers/Package.swift`: Swift tools 5.9, macOS 14+, HelperCore와 3개 executable target.
- `Makefile`: `make build`는 Go + Swift release helpers, `make test`는 Go + Swift tests.
- `.github/workflows/ci.yml`: macOS runner에서 build, Go tests, Swift tests.
- 기존 `.analysis/AI_CONTEXT.md`는 8월 스냅샷이며 과거 작업지시서의 Go/macOS 버전을 그대로 재사용하면 안 된다.

## 실제 연결점

| 기존 파일/심볼 | 확인한 책임 | 계획에 주는 영향 |
|---|---|---|
| `cmd/break-reminder/pomodoro.go`, `newPomodoroCmd` | on/off/status, 시간 옵션, config 저장 후 cycle reset | GUI는 기존 명령을 사용하고 오류·전환 경계를 보완 |
| 같은 파일 `resetCurrentCycle` | work 상태만 초기화, `state.Update` 오류 무시 | GUI에 성공 표시하기 전 오류 전파 필요 |
| `internal/timer/timer.go`, `Tick`, `tickWork`, `startBreak` | 활동 감지·gap·pause·회차·긴 휴식 | 포모도로는 벽시계 타이머가 아님 |
| `cmd/break-reminder/session_cmd.go` | 수동 start/stop, 근무시간 밖 시작 | 포모도로 모드 선택만으로 세션 시작된다고 가정 금지 |
| `cmd/break-reminder/check.go`, `runCheck` | 근무 정책 gate, state 저장 후 action | 독립 타이머를 gate 아래에 넣으면 안 됨 |
| `internal/launchd/launchd.go`, `generateTimerPlist` | `check`를 StartInterval 60초로 실행 | 새 worker 필요, 기존 check는 그대로 유지 |
| `cmd/break-reminder/daemon.go` | cfg interval마다 runCheck | 이 명령도 기본적으로 초 단위 타이머 worker가 아님 |
| `internal/state/state.go`, `Update` | flock 안에서 load/mutate/atomic rename | 별도 store 구현 시 참고하되 private 함수 직접 재사용 불가 |
| `internal/notify/notify_darwin.go` | terminal-notifier 실행, 고정 group | 새 완료 알림에 event별 group 필요 |
| `helpers/Sources/MenuBarApp/main.swift` | 1초 refresh, AppKit 메뉴, reset/force 직접 쓰기 | 새 조작은 비동기 CLI, 새 store 직접 쓰기 금지 |
| `helpers/Sources/HelperCore/MenuBarTimer.swift` | common + eventTracking run loop | 메뉴를 열어둔 상태에서도 표시 갱신 재사용 |
| `helpers/Sources/DashboardApp/DashboardViewModel.swift` | state/config 읽기, 동기 runCLI, 일부 직접 쓰기 | 테스트 가능한 비동기 command client 추가 |
| `helpers/Sources/DashboardApp/DashboardAppMain.swift` | 360×600 고정 창, Q/R/B local key monitor | 도구 뷰 scroll 및 입력 중 단축키 차단 |
| `helpers/Sources/DashboardApp/TimerTabView.swift` | 일일 통계/시스템 정보/Reset/Force Break | 집중/타이머 하위 선택 영역으로 확장 |
| `helpers/Tests/HelperCoreTests/DashboardViewModelTests.swift` | 이름과 달리 HelperCore 계산 함수 테스트 | 실제 DashboardViewModel UI 테스트로 오인 금지 |

## 리뷰 후 추가 확인 (2026-09-28)

- 문서 8개는 작업 트리에 실제 존재하나 `git ls-files`에는 없다. 파일 미작성과 미커밋을 구분한다. 이번 리뷰 반영 기록까지 9개를 동일 문서 커밋으로 전달해야 한다.
- `.analysis/AI_CONTEXT.md`와 다수 분석 결과는 tracked. cache와 두 최근 분석 보고서 등은 untracked다. 디렉터리 전체를 미추적으로 표현하지 않는다.
- `internal/autoupdate/autoupdate.go:CheckAndUpgrade`는 brew update/outdated/upgrade를 호출한다. `cmd/break-reminder/update.go`가 이후 `launchd.RestartRuntime`을 호출하지만, `internal/launchd/launchd.go:RestartRuntime`은 기존 timer/menu job을 load할 뿐 새 plist를 만들지 않는다.
- `Formula/break-reminder.rb:post_install`은 service install 안내만 출력한다. 구버전 updater로 첫 업그레이드될 때 새 worker 자동 설치는 현재 경로로 보장되지 않는다.
- `internal/doctor/doctor.go`는 이미 notifier Send 오류를 fail로 보고한다. 보완 대상은 실패 감지 신설이 아니라 worker 상태·알림 fallback·복구 안내의 구체화다.

## 유지할 컨벤션

- Go: `internal/<domain>`에 순수 로직, `cmd/break-reminder`에 Cobra 조립, snake_case 파일명, `%w` 오류 래핑, 표준 testing과 임시 경로·시간 주입.
- Swift: PascalCase 파일명, HelperCore에 순수 모델/표현·공유 로직, SwiftUI Dashboard와 AppKit MenuBar 어댑터 분리. 테스트 target은 현재 HelperCore에만 의존한다.
- 새 command client는 `Process.arguments` 배열을 사용하고 shell 문자열로 실행하지 않는다. helper 탐색은 기존 명시적 경로 정책을 따른다.
- 문서는 기존 `docs/plans/`에 기능 접두사 `time-tools-`로 추가한다. 기존 pause/settings 로드맵은 수정하지 않는다.

## 이미 존재하지만 그대로 쓰면 위험한 패턴

1. Swift `resetTimer`/`forceBreak`는 기존 state를 다시 직렬화한다. 새 JSON store를 별도로 두어 새 도구 데이터가 이 경로에 의해 유실되지 않게 한다. 기존 state writer 전체 제거는 이번 MVP의 필수 작업이 아니다.
2. 기존 CLI `reset`은 `state.New()`를 저장하므로 GUI의 일일 통계 보존 reset과 의미가 다르다. GUI를 단순히 `reset` 호출로 바꾸면 회귀다.
3. `break` CLI는 가이드 활동 명령이며 강제 휴식 상태 전환 API가 아니다. 같은 이름을 재사용하지 않는다.
4. config와 state는 별도 파일이다. 포모도로 on/off를 원자적 트랜잭션이라고 표현하지 않는다. 저장 후 reset 실패를 보고하고 실제 상태를 다시 읽어야 한다.
5. 시스템 알림 프로세스 성공은 사용자가 배너를 보았다는 증거가 아니다. UI 완료 상태와 알림 전송 결과를 분리한다.

## 검증 방법

기존 검증 명령: `go test ./...`, `swift test --package-path helpers`, `make test`, `make build`. 새 기능은 먼저 관련 package/filter로 RED/GREEN을 확인한 뒤 최종 변경에서 전체 검증한다.

이번 계획 작성 중에는 소스를 수정하지 않았으므로 제품 테스트를 실행하지 않는다. 구현자는 시작 시 baseline 실패 여부를 따로 기록해야 한다. 테스트/수동 QA 명령은 [수용 기준](time-tools-acceptance.md)을 따른다.
