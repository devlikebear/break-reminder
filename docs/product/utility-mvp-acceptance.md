# Start One Guided Break — MVP 수용 기준

- 기준 문서: `docs/product/utility-mvp-prd.md`
- 단계: BreakScreenApp 통합 구현 및 Swift 자동 증거 기록 완료; 권한 부여 후 read-only 수동 UI는 부분 통과했으나 입력·VoiceOver·123초 QA는 headless 승인 정책으로 BLOCKED
- 판정 원칙: P0는 모두 통과해야 구현 완료다. 자동화할 수 없는 AppKit/VoiceOver 항목은 명시된 수동 관찰 결과와 실행 환경을 기록한다.
- 실제 증거 갱신: 2026-08-19 자동 실행 결과는 아래와 `utility-mvp-implementation.md`에 기록했다. 실행하지 않은 항목은 `NOT RUN`으로 남긴다.

## 1. AC 요약

| AC-ID | 우선순위 | 사용자 결과 | 판정 방식 | 2026-08-19 실제 증거 |
|---|---|---|---|---|
| AC-01 | P0 | block 휴식 진입 시 한 활동의 ready 화면을 본다 | Swift unit + 수동 UI | Swift unit PASS; 600초 ready/no-auto-start와 short-state read-only UI PASS; 정확한 123초 프레임·multidisplay 미검증 |
| AC-02 | P0 | 한 번 시작해 정확히 120초 진행하고 완료를 확인한다 | Swift unit + 123초 실런타임 | Swift unit PASS; 123초 실런타임 `BLOCKED` — CuaDriver mutation approval timeout으로 Start 불가 |
| AC-03 | P0 | 무시·가이드 취소·Esc·기존 Skip이 예측 가능하다 | Swift unit + 수동 UI | no-auto-start UI와 enabled Skip read-only PASS; cancel/restart/Esc/5초 focus는 입력 승인 부재로 `BLOCKED` |
| AC-04 | P0 | 짧은 잔여 시간·재시작·기존 상태가 안전하다 | Swift/Go unit | Swift guard/fresh-session unit 및 Go 회귀 PASS |
| AC-05 | P0 | helper/인자 실패가 기존 fallback/default로 복구된다 | Go/Swift unit | Swift ArgsParser 및 Go breakscreen 회귀 PASS |
| AC-06 | P0 | 키보드와 VoiceOver로 상태·조작을 이해할 수 있다 | 수동 접근성 QA + build | Swift build 및 exact AX button 노출 PASS; keyboard/VoiceOver/Reduce Motion/Increase Contrast는 mutation 승인·음성 경로 부재로 `BLOCKED` |
| AC-07 | P0 | 60초 launchd 실행 계약과 state/config/CLI/사용자 자산이 보존된다 | source contract + 독립 Go/Swift unit + diff + 수동 launchd QA | blocking/missed-firing source contract, Swift 60-tick, Go suite/race, diff 보호 PASS; 실제 launchd 재개 `NOT RUN` |
| AC-08 | P0 | 전체 Go/Swift 회귀와 빌드가 통과한다 | 전체 자동화 | Go/Swift tests, race, vet, release build, diff PASS |

## 2. 상세 수용 기준

### AC-01 — 결정적인 ready 진입

우선순위: P0

Given
- `break_screen_mode=block` 경로가 `BreakScreenApp`을 실행하고,
- `BreakScreenArgs(duration: 600, skipAfter: 120, todayWorkMin: 60, todayBreakMin: 10)`을 받았을 때

When
- primary window가 표시되고 1초 tick이 한 번 이상 지나도 사용자가 시작하지 않으면

Then
- 무작위 활동이 아니라 `2분 동안 서서 목과 어깨를 풀어보세요` 한 카드만 보인다.
- `시작`이 enabled이며 기본 동작이다.
- phase는 `.ready`를 유지하고 자동 시작하지 않는다.
- 기존 전체 휴식 카운트다운·진행률·당일 통계·비활성 Skip·Esc 안내가 계속 보인다.
- secondary display는 기존 비상호작용 휴식 표면을 유지한다.

테스트 레벨·fixture
- Unit: `GuidedBreakSession()` 초기 phase와 no-auto-start.
- Manual AppKit: 600초/120초 fixture, 다중 display가 있으면 primary/secondary 확인.

명령
- `cd helpers && swift test --filter GuidedBreakSessionTests`
- `make build && bin/break-screen --duration 600 --skip-after 120 --work-min 60 --break-min 10`

예상 증거 위치
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testInitialPhaseIsReady`
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testTickDoesNotAutoStartReadySession`
- `docs/product/utility-mvp-implementation.md#AC-01`

### AC-02 — 시작→120초→완료→종료

우선순위: P0

Given
- ready phase이고 남은 전체 휴식이 600초일 때

When
- 사용자가 `시작`을 한 번 실행하고 1초 tick을 123번 전달하면

Then
- 시작 직후 `.running(remainingSeconds: 120)`이다.
- 60번째 tick 직후 `.running(remainingSeconds: 60)`이고 현재 단계 문구와 `01:00`이 보인다.
- 120번째 tick 직후 `.completed(remainingDisplaySeconds: 3)`이며 완료 문구가 보인다.
- 완료 phase가 된 순간 `2분 스트레칭을 완료했습니다.` announcement를 한 번만 요청한다.
- 추가 3초 후 `tick()`이 `.dismiss`를 반환하고 overlay가 닫힌다.
- 그 과정에서 새 프로세스, 터미널, 네트워크를 열지 않는다.

테스트 레벨·fixture
- Unit: virtual 1초 tick 123개; 120/80/40/1/0 경계의 phase와 단계 문구.
- Manual real runtime: `duration=130`, `skipAfter=120`; 시작을 5초 안에 누르고 stopwatch로 60초·120초·123초 관찰.

명령
- `cd helpers && swift test --filter GuidedBreakSessionTests`
- `make build && bin/break-screen --duration 130 --skip-after 120`

예상 증거 위치
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testStartRunsForExactly120TicksThenShowsCompletionForThreeTicks`
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testInstructionTextAtAllBoundaries`
- `docs/product/utility-mvp-implementation.md#AC-02`

### AC-03 — 무시·취소·Esc·기존 Skip

우선순위: P0

Given
- ready 또는 running phase인 overlay가 표시됐을 때

When / Then
1. 시작을 누르지 않으면 phase는 ready이고 기존 전체 휴식 타이머만 감소한다.
2. running에서 `가이드 취소`를 누르면 session은 ready로 돌아가고 overlay와 전체 휴식 타이머는 계속된다. 남은 전체 휴식이 123초 이상이면 재시작할 수 있다.
3. 어느 phase에서든 Esc를 누르면 즉시 overlay가 닫힌다.
4. `elapsed < args.skipAfter`이면 기존 Skip은 disabled이고 focus 대상이 아니다.
5. `elapsed >= args.skipAfter`이면 `Skip Break`가 enabled되고 실행 시 즉시 overlay가 닫힌다.
6. 가이드 기능은 skipAfter를 줄이거나 Go snooze/state를 변경하지 않는다.

테스트 레벨·fixture
- Unit: start→tick→cancel→ready→restart.
- Manual AppKit: `duration=180`, `skipAfter=5` fixture로 Esc와 disabled/enabled 전이를 각각 확인.

명령
- `cd helpers && swift test --filter GuidedBreakSessionTests`
- `make build && bin/break-screen --duration 180 --skip-after 5`

예상 증거 위치
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testCancelReturnsRunningSessionToReadyAndAllowsRestart`
- `docs/product/utility-mvp-implementation.md#AC-03`

### AC-04 — 잔여 시간 guard·재시작·기존 상태 호환

우선순위: P0

Given / When / Then
1. `availableBreakSeconds=122`에서 `start`하면 `false`를 반환하고 ready를 유지한다.
2. `availableBreakSeconds=123`에서 `start`하면 `true`를 반환하고 running(120)이 된다.
3. ready 중 전체 휴식 remaining이 123 미만이 되면 Start가 disabled되고 `이번 휴식에는 2분이 남지 않았어요.`가 텍스트로 보인다.
4. helper를 재실행하면 이전 guided phase를 복원·추정하지 않고 ready에서 시작한다. 기존 CLI의 계산된 `--duration`만 따른다.
5. 기존 state 파일에서 `EnterBreak`는 `Mode=break`, `BreakStart`, `SnoozeUntil=0`, `WorkSeconds=0`, `LastBreakWarningBucket=0` 계약을 유지한다.
6. 전체 휴식 remaining이 running 중 0이면 guided completion으로 기록하지 않고 overlay를 닫는다.

테스트 레벨·fixture
- Swift unit: 122/123 boundary, fresh session, parent timeout precedence를 UI adapter 또는 분리 가능한 순수 로직으로 테스트.
- Go unit: 기존 EnterBreak와 break remaining 계산 회귀.

명령
- `cd helpers && swift test --filter GuidedBreakSessionTests`
- `go test ./internal/state ./internal/timer ./internal/breakscreen`

예상 증거 위치
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testStartRequiresActivityAndCompletionBudget`
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testNewSessionDoesNotRestorePreviousPhase`
- `internal/state/state_test.go::TestEnterBreakResetsWarningBucket`
- `docs/product/utility-mvp-implementation.md#AC-04`

### AC-05 — helper·인자 실패 복구

우선순위: P0

Given / When / Then
1. block mode에서 `break-screen` helper를 찾지 못하면 panic/hang 없이 기존 notification fallback을 호출하고 반환한다.
2. helper 프로세스가 non-zero로 종료해도 Go의 저장된 break state를 되돌리거나 새 state를 쓰지 않고 warning 경로로 반환한다.
3. `--duration` 등 숫자 인자가 누락/비정상이면 기존 `BreakScreenArgs` 기본값(`duration=600`, `skipAfter=120`, 통계 0)을 유지한다.
4. 새 guided flag나 새 parser fallback은 존재하지 않는다.

테스트 레벨·fixture
- Go unit: helper lookup/runner seam의 not-found 및 non-zero fixture. 제품 코드 seam이 불필요하면 동일 package 테스트 stub을 사용한다.
- Swift unit: 기존 ArgsParser invalid/missing/default fixture.

명령
- `go test ./internal/breakscreen`
- `cd helpers && swift test --filter ArgsParserTests`

예상 증거 위치
- `internal/breakscreen/breakscreen_test.go::TestShowBlockWithoutHelperFallsBackAndReturns`
- `internal/breakscreen/breakscreen_test.go::TestOverlayHelperFailureDoesNotMutateState`
- `helpers/Tests/HelperCoreTests/ArgsParserTests.swift::{testDefaults,testInvalidNumber,testMissingValue}`
- `docs/product/utility-mvp-implementation.md#AC-05`

### AC-06 — 키보드·VoiceOver·비색상 접근성

우선순위: P0

Given
- macOS VoiceOver를 켜거나 키보드만 사용하는 사용자가 ready/running/completed 화면에 있을 때

When / Then
1. ready에서 Return 또는 Space 한 번으로 `시작`을 실행한다.
2. focus 순서는 현재 phase의 주 동작(`시작` 또는 `가이드 취소`) 다음 활성화된 `Skip Break`이며 disabled Skip은 제외한다.
3. 시작 control은 label `2분 목과 어깨 스트레칭 시작`, hint `같은 화면에서 2분 가이드를 시작합니다.`를 읽는다.
4. running은 `목과 어깨 스트레칭`, 남은 시간, 현재 단계를 value 또는 결합 label로 읽는다.
5. 취소 control은 label `가이드 취소`, hint `가이드를 멈추고 휴식 화면으로 돌아갑니다.`를 읽는다.
6. completed 진입 시 `2분 스트레칭을 완료했습니다.`를 한 번 announcement한다.
7. ready/running/completed는 색이나 progress bar 없이도 title·시간·단계·완료 텍스트로 구분된다.
8. Reduce Motion 활성화 상태에서도 기능·정보가 동일하고 새 필수 애니메이션이 없다.

테스트 레벨·fixture
- Manual accessibility QA: VoiceOver Utility/키보드, Reduce Motion on/off.
- Build: AppKit API와 accessibility 속성 컴파일 확인.

명령
- `make build`
- 수동: 시스템 설정에서 VoiceOver 및 Reduce Motion을 각각 켜고 `bin/break-screen --duration 180 --skip-after 5` 실행

예상 증거 위치
- `docs/product/utility-mvp-implementation.md#AC-06`에 macOS 버전, 입력 방식, 실제 낭독 문자열, focus 순서, pass/fail 기록

### AC-07 — 60초 launchd 실행 계약·상태/API/자산 무변경

우선순위: P0

Given
- Go state가 이미 `EnterBreak`로 저장됐고 `showOverlay`의 `cmd.Run()`이 running Swift helper의 종료를 기다릴 때

When
- helper 실행 중 하나 이상의 60초 `StartInterval` firing이 지나고, 이후 helper가 종료되면

Then
- 실행 중 firing에서는 동일 launchd job의 새 Go check가 동시에 시작되지 않고 해당 firing은 missed된다.
- blocking check가 helper 종료 후 반환하고, 다음 `StartInterval`에서 새 Go check가 재개된다. missed firing의 catch-up을 기대하지 않는다.
- 그동안 저장된 Go state는 `Mode=break`와 원래 `BreakStart`를 유지한다. 이후 check의 경과 시간·통계 처리는 기존 Go 정책을 따른다.
- Swift의 독립 60-tick 모델은 running(60)이며 Go state/config/history/log 파일을 쓰지 않는다. 이 unit 결과를 동시 launchd tick의 증거로 사용하지 않는다.
- Go `State` schema, Load/Save 양방향 key, config key, `BreakScreenArgs`, `showOverlay` CLI flag 목록은 변경되지 않는다.
- `EnterBreak`를 우회하는 새 break 전환이 없다.
- 루트 PNG, hamster/vector assets, `.analysis/**`, `.serena/**`는 수정·삭제·이동·스테이징되지 않는다.
- 메뉴바, Dashboard, notify mode, TUI activity 동작은 변경되지 않는다.

테스트 레벨·fixture
- Source contract: `cmd.Run()` blocking 경로와 macOS `launchd.plist(5)`의 running-job missed-firing 규칙.
- Go unit: launchd와 독립적으로 break state에서 elapsed=60, idle active/idle 양쪽 fixture; State serialize/load round trip.
- Swift unit: running 60 tick.
- Source/diff contract inspection.
- Manual launchd: 실제 LaunchAgent에서 helper 종료 뒤 다음 interval 재개와 state 선저장을 관찰하며, 실행 전까지 `NOT RUN`으로 둔다.

명령
- `go test ./internal/timer ./internal/state ./internal/breakscreen ./cmd/break-reminder`
- `cd helpers && swift test --filter GuidedBreakSessionTests`
- `git diff --name-only`
- `git diff --cached --name-only`
- `git status --short --branch`

예상 증거 위치
- `internal/timer/timer_test.go`의 60초 break 누적 fixture(기존 또는 명시적 신규 test)
- `internal/state/state_test.go`의 Load/Save round-trip 및 `TestEnterBreakResetsWarningBucket`
- `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift::testSixtyTicksLeavesSixtySeconds`
- `docs/product/utility-mvp-implementation.md#AC-07`

### AC-08 — 전체 회귀·빌드

우선순위: P0

Given
- 구현 diff가 준비됐을 때

When
- 아래 품질 게이트를 깨끗한 명령으로 실행하면

Then
- Go 전체 test와 vet, Swift 전체 test, release helper+Go build, diff whitespace 검사가 모두 exit 0이다.
- 실패한 테스트를 삭제·skip·완화해 통과시키지 않는다.
- `helpers/Tests/HelperCoreTests`의 기존 테스트 수와 기존 Go 패키지 테스트가 회귀하지 않는다.

명령
- `go test ./...`
- `(cd helpers && swift test)`
- `go vet ./...`
- `make build`
- `git diff --check`

예상 증거 위치
- `docs/product/utility-mvp-implementation.md#AC-08`에 명령, exit code, Go package 결과, Swift test count/failure count 기록

## 3. 구현 매핑 체크리스트

모든 P0 AC는 아래 세 칸을 충족해야 한다.

| AC-ID | 구현 지점 | 테스트 지점 | 객관 판정 결과 |
|---|---|---|---|
| AC-01 | `BreakScreenApp/main.swift`, `GuidedBreakSession.swift` | `GuidedBreakSessionTests`, 수동 UI | 고정 card/ready/no auto-start |
| AC-02 | `GuidedBreakSession.swift`, AppKit timer adapter | boundary unit tests, 123초 수동 QA | 60초 running(60), 120초 completed(3), 123초 dismiss |
| AC-03 | session cancel + 기존 Esc/Skip handler | cancel/restart unit, 수동 key/button | ready 복귀 또는 즉시 dismiss; state write 없음 |
| AC-04 | start guard + parent timeout precedence | 122/123/fresh session, Go state tests | 짧은 잔여 시간 거부, 재실행 ready, 기존 state 보존 |
| AC-05 | 기존 breakscreen fallback + ArgsParser | Go failure fixture, ArgsParserTests | fallback/default 후 정상 반환 |
| AC-06 | AppKit controls/accessibility properties | 수동 VoiceOver/키보드/Reduce Motion | 지정 문자열·focus·비색상 구분 확인 |
| AC-07 | 기존 blocking Go check/timer/state + Swift volatile session | launchd source contract + 독립 60초 Go/Swift tests + diff + 수동 launchd QA | 실행 중 firing missed, helper 종료 뒤 다음 interval 재개, schema/API/assets 무변경; 실제 launchd 관찰은 `NOT RUN` |
| AC-08 | 전체 저장소 | full test/vet/build/diff | 모든 명령 exit 0 |

## 4. Definition 및 구현 증거 검토 결과

- 핵심 AC 수: 8개(요구 범위 5–8 충족)
- P0 구현 지점·테스트 지점·판정 결과 매핑: 완료
- 외부 사실 미검증 항목: 실제 사용자 완료율, 최적 활동/길이, block-mode 비율
- 원격 텔레메트리: 명시적 비범위
- Swift TDD/통합 상태: handoff의 RED는 `GuidedBreakSession` 타입 부재로 인한 compile error였으며, 현재 filtered GREEN은 10 tests/0 failures다.
- Swift 전체 회귀: `cd helpers && swift test`는 101 tests/0 failures, `cd helpers && swift build`는 exit 0이다.
- 수동 UI 재시도: 2026-08-19 14:38–14:57 KST에 awake 캡처와 exact PID/window AX bind로 600초 ready/no-auto-start, short-state, enabled Skip, 기본 대비/잘림을 실제 관찰했다. CuaDriver mutation은 headless approval timeout으로 거부돼 keyboard, VoiceOver, Reduce Motion/Increase Contrast 전환, 123초 실제 runtime은 계속 `BLOCKED / NOT RUN`이다. 상세 증거는 `utility-mvp-manual-qa.md`에 기록했다.
- 현재 판정: Go/Swift 자동 품질 게이트는 모두 통과했다. 수동 UI·접근성·123초 실런타임 P0는 독립 검토 전까지 완료로 판정하지 않는다.

## 4.1. 2026-08-20 `--yolo` 수용 재판정

사용자가 이 한 번의 QA에 한해 `bin/break-screen` fixture와 CuaDriver exact PID/window mutation을 명시 승인했다. 상세 명령·환경·AX 문자열·epoch는 `utility-mvp-manual-qa.md`에 기록했다.

| AC | 실제 재검증 | 최신 판정 |
|---|---|---|
| AC-01 | 600초 fresh ready capture와 기존 9분 no-auto-start 증거, 고정 card/통계/disabled Skip 유지 | PASS (secondary display N/A) |
| AC-02 | Start/Return/Space 실제 실행, running `01:59`, exact `01:00`, 120초 completed, completed→exit 약 2.766초 | PASS — `02:00` 단일 프레임만 미포착 |
| AC-03 | Cancel→ready→Space restart, ready/running/completed Esc, skip-after 전 disabled/후 enabled 및 Skip exit | PASS |
| AC-04 | 122에서 disabled/short copy, 124 fixture의 경계 tick 123에서 Return 수락 후 running, fresh helper ready | PASS |
| AC-05 | 이번 QA에서 parser/fallback 코드는 변경하지 않음; 기존 자동 회귀 PASS 유지 | PASS |
| AC-06 | Return/Space와 Tab/Shift-Tab delivery, 지정 AX label/value/help 및 비색상 텍스트 상태 PASS. VoiceOver 실제 음성·announcement event count, focus ring/`AXFocused`, Reduce Motion/Increase Contrast는 관찰 환경 미충족 | PARTIAL PASS / N/A 항목 명시 |
| AC-07 | 보호 자산/CLI/state/config/launchd 코드 무변경, staged 없음. 실제 LaunchAgent firing은 승인된 fixture 범위 밖이라 기존 source/자동 계약만 재사용 | PASS(source/자동/diff) / manual launchd NOT RUN |
| AC-08 | 시작 전 `make build` PASS; 종료 시 `make test`, `git diff --check`, staged/protected-asset 검사를 별도 기록 | 종료 게이트 결과는 구현 증거 문서 참조 |

P0의 실제 Guided Break 핵심 흐름은 더 이상 mutation approval로 차단되지 않는다. 남은 항목은 제품 결함이 아니라 이 환경에서 실제 음성/announcement event/focus 속성, 접근성 설정 변형, 다중 display, 실제 LaunchAgent firing을 관찰하지 못한 N/A 또는 별도 수동 범위다.

## 5. 실제 증거 위치

- TDD RED/GREEN, AC별 자동/수동 판정, 명령 결과: `docs/product/utility-mvp-implementation.md`
- 통합 구현: `helpers/Sources/BreakScreenApp/main.swift`
- pure session과 10개 경계 테스트: `helpers/Sources/HelperCore/GuidedBreakSession.swift`, `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift`
