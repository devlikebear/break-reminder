# Start One Guided Break — 구현 증거

- 기록일: 2026-08-19 (Asia/Seoul)
- 구현 범위: `BreakScreenApp/main.swift`의 승인된 guided break UI adapter와 사용자 문서
- 상태: Swift 자동 검증 통과; 권한 부여 후 read-only 수동 UI는 부분 통과했으나 입력·VoiceOver·123초 실런타임은 headless mutation 승인 timeout으로 `BLOCKED / NOT RUN`
- 범위 보호: `GuidedBreakSessionTests.swift`를 먼저 추가해 RED를 확인한 뒤 `GuidedBreakSession.swift`를 구현했다. stage/commit은 하지 않았다.

## TDD 증거

### RED — 직접 재현

- 명령: `cd helpers && swift test --filter GuidedBreakSessionTests`
- 결과: exit code 1. `GuidedBreakSession` 구현이 없어 test target compile이 실패했고 핵심 diagnostic은 `cannot find 'GuidedBreakSession' in scope`였다.
- 실패 원인 확인: 테스트 오타나 환경 오류가 아니라 새 모델 API의 부재 때문에 실패했다.

### GREEN — 현재 작업에서 확인

명령:

```bash
cd helpers && swift test --filter GuidedBreakSessionTests
```

실제 결과:

- 실행 시각: 2026-08-19 10:29:49 KST
- exit code: 0
- `GuidedBreakSessionTests`: 10 tests, 0 failures, 0 unexpected
- 검증 범위: initial ready/no-auto-start, 122/123 guard, phase-only start, 60/120/123 ticks, instruction boundaries, cancel/restart/no-op, fresh session

## BreakScreenApp 통합

`helpers/Sources/BreakScreenApp/main.swift`에서 다음을 연결했다.

- 무작위 activity 문구를 고정 높이 ready/running/completed 카드로 교체했다.
- 기존 한 개의 1초 `Timer`가 전체 countdown과 `GuidedBreakSession.tick()`을 함께 구동한다. 별도 timer는 추가하지 않았다.
- Go `check`는 `cmd.Run()`에서 이 helper의 종료를 기다린다. 따라서 helper 실행 중 동일 launchd job의 60초 firing은 missed되며 Swift timer와 동시 Go check가 실행된다고 주장하지 않는다.
- 각 tick에서 전체 `remaining <= 0`을 먼저 처리해 parent timeout이 guided completion보다 우선한다.
- Start는 session의 123초 budget guard를 사용하고, Cancel은 overlay를 닫지 않은 채 ready로 돌아간다.
- 기존 Esc local monitor, elapsed 기반 Skip, secondary display의 `☕ Break Time`, 전체 countdown/progress와 선택적 당일 통계를 유지했다.
- ready/running control을 44pt로 만들고, Return default action, Space의 AppKit button 동작, phase별 first responder와 key-view loop를 연결했다. disabled Start/Skip은 first responder를 거부한다.
- 전체/guide 시간, 진행률, 카드, Start/Cancel/Skip에 AppKit accessibility label/value/help를 지정했다.
- completed 진입의 `.phaseChanged`에서만 `2분 스트레칭을 완료했습니다.` announcement를 요청하고 boolean guard로 한 번만 게시한다.

## AC-01 — 결정적인 ready 진입

- 자동 증거: `testInitialPhaseIsReady`, `testTickDoesNotAutoStartReadySession` PASS.
- 소스 증거: fixed ready card, default Start, overall timer/progress/stats/Skip/Esc, unchanged secondary branch가 `swift build`로 컴파일됐다.
- 수동 primary/secondary UI 확인: `NOT RUN`.
- 판정: 자동 범위 PASS / 전체 AC 보류.

## AC-02 — 시작→120초→완료→종료

- 자동 증거: `testStartRunsForExactly120TicksThenShowsCompletionForThreeTicks`, `testInstructionTextAtAllBoundaries`, `testSixtyTicksLeavesSixtySeconds` PASS.
- 소스 증거: overall timeout 검사 뒤 session tick, completed phase transition에서 1회 announcement, `.dismiss`에서 overlay 종료가 컴파일됐다.
- stopwatch를 사용한 123초 실제 runtime QA: `NOT RUN`.
- 판정: 모델 자동 범위 PASS / 실제 시간·화면 관찰 보류.

## AC-03 — 무시·취소·Esc·기존 Skip

- 자동 증거: `testTickDoesNotAutoStartReadySession`, `testCancelReturnsRunningSessionToReadyAndAllowsRestart`, `testCancelIsNoOpOutsideRunningPhase` PASS.
- 소스 증거: 기존 Esc keyCode 53 monitor와 elapsed/skipAfter 전이를 유지했고 Cancel만 ready render로 연결했다.
- Esc, disabled/enabled Skip, Cancel/restart 수동 UI QA: `NOT RUN`.
- 판정: 자동 범위 PASS / 전체 AC 보류.

## AC-04 — 잔여 시간 guard·재시작·기존 상태 호환

- 자동 증거: `testStartRequiresActivityAndCompletionBudget`, `testNewSessionDoesNotRestorePreviousPhase` PASS.
- 소스 증거: Start enabled 조건과 model start 모두 120+3초를 사용하고, `tick()`은 전체 timeout을 먼저 처리한다.
- 122/123초 실제 UI와 helper 재실행: `NOT RUN`.
- Go state/timer/breakscreen/cmd 회귀와 전체 Go suite: PASS.
- 판정: 자동 범위 PASS / 실제 UI 관찰 보류.

## AC-05 — helper·인자 실패 복구

- 자동 증거: 전체 Swift suite 안에서 `ArgsParserTests` 11 tests/0 failures.
- 변경 범위: guided flag/parser 또는 Go helper/fallback 코드는 변경하지 않았다.
- 기존 `internal/breakscreen` 회귀와 전체 Go suite: PASS.
- 판정: 자동 회귀 PASS.

## AC-06 — 키보드·VoiceOver·비색상 접근성

- 빌드 증거: AppKit default button, first responder/key-view loop, accessibility role/label/value/help, announcement API가 `swift build`에서 컴파일됐다.
- 화면 상태는 title/time/instruction/completion text로 구분하며 새 animation은 추가하지 않았다.
- keyboard-only, VoiceOver 실제 낭독, announcement 횟수, Reduce Motion, Increase Contrast QA: `NOT RUN`.
- 판정: compile PASS / 수동 접근성 판정 보류.

## AC-07 — 60초 launchd 실행 계약·상태/API/자산 무변경

- 자동 증거: `testSixtyTicksLeavesSixtySeconds` PASS.
- source contract 증거: `showOverlay`는 `cmd.Run()`으로 helper 종료까지 block하고 macOS `StartInterval`은 실행 중 동일 job의 firing을 missed한다. 저장된 break state는 helper 시작 전에 기록되며, 다음 Go check는 helper 종료 뒤 다음 interval에서 재개된다.
- diff 증거: 이번 통합은 Go state/config/CLI, root PNG, hamster/vector, `.analysis/**`, `.serena/**`를 편집하지 않았다. 이들 중 일부는 시작 전부터 dirty였으며 그대로 보호했다.
- staged diff: 없음.
- Go의 독립 60초/state 관련 패키지와 전체 Go suite/race: PASS. 이는 overlay 중 동시 Go tick의 증거가 아니다.
- 실제 LaunchAgent에서 helper 종료 뒤 다음 interval 재개, state 선저장, multidisplay 동작: `NOT RUN`.
- 판정: source/자동/diff 범위 PASS / 실제 launchd·multidisplay 관찰 보류.

## AC-08 — 전체 회귀·빌드

실제 실행:

| 명령 | 결과 |
|---|---|
| `cd helpers && swift test --filter GuidedBreakSessionTests` | PASS — 10 tests, 0 failures |
| `cd helpers && swift test` | PASS — 101 tests, 0 failures |
| `cd helpers && swift build --product BreakScreenApp` | PASS — exit 0 |
| `cd helpers && swift build` | PASS — exit 0 |
| `git diff --check` | PASS — whitespace error 없음 |
| `go test ./...` | PASS — 전체 패키지 exit 0 |
| `go test -race ./...` | PASS — 전체 패키지 exit 0 |
| `go vet ./...` | PASS — exit 0 |
| `make build` | PASS — release Swift helpers + Go binary exit 0 |

- 판정: repository-wide 자동 품질 게이트 PASS. 실제 UI/접근성 관찰은 별도 수동 QA로 남는다.

## 수동 QA 상태

첫 시도는 asleep display와 권한 문제로 차단됐다. 14:38 KST 권한 부여 후 재시도에서는 `hermes computer-use doctor`가 Accessibility/Screen Recording/AX healthy를 보고했고, `screencapture`와 exact PID/window CuaDriver capture가 성공했다. 600초 ready/no-auto-start, short-state, enabled Skip, 기본 대비와 잘림은 실제 화면에서 확인했다. 다만 CuaDriver mutation은 headless approval timeout으로 거부됐고 VoiceOver도 실행되지 않아 상호작용·음성·123초 실런타임은 관찰하지 않았다. 상세 명령과 환경은 `utility-mvp-manual-qa.md`에 기록했다.

| QA | 범위 | 상태 |
|---|---|---|
| QA-01 | 600초 ready, 122초 short, no-auto-start, primary/secondary UI | `PARTIAL PASS` — primary ready/no-auto-start와 short-state 실제 관찰; 정확한 123초 프레임 미확보, secondary는 단일 display라 N/A |
| QA-02 | stopwatch 기반 60/120/123초 실제 runtime과 자동 종료 | `BLOCKED / NOT RUN` — Start mutation approval timeout |
| QA-03 | Cancel/restart, Esc, disabled/enabled Skip UI | `PARTIAL / BLOCKED` — no-auto-start와 enabled Skip 시각 확인; Cancel/Esc/5초 focus 입력 불가 |
| QA-04 | keyboard focus/Return/Space/Tab과 VoiceOver 실제 낭독·1회 announcement | `BLOCKED / NOT RUN` — mutation approval timeout, VoiceOver off |
| QA-05 | Reduce Motion, Increase Contrast, multidisplay | `PARTIAL PASS` — 기본 설정 대비·잘림 확인; 설정 on/focus ring 미검증, 디스플레이 1대라 multi-display N/A |
| QA-06 | helper failure/restart, state 선저장, helper 종료 뒤 launchd interval 재개 | `PARTIAL / BLOCKED` — 재실행마다 fresh ready는 관찰; running 강제 종료·실제 launchd interval 재개는 NOT RUN |

따라서 ready/short 기본 UI 배치 외 VoiceOver 발화, focus ring, 다중 display, timer scheduling 오차 또는 123초 종료 동작을 확인했다고 주장하지 않는다. 남은 차단 해제에는 지정 PID/window에 한정된 CuaDriver mutation 사전 승인과 실제 VoiceOver 음성 청취 경로가 필요하다.

## 2026-08-20 실제 CuaDriver Guided Break 증거

- 범위: 사용자 1회 `--yolo` 승인에 따라 실제 `bin/break-screen` fixture와 각 fixture의 exact `(pid, window_id)`만 캡처·조작했다. 제품 코드·테스트·README·CHANGELOG·`.analysis`·`.serena`·루트 PNG·git index는 편집하지 않았다.
- 사전 조건: `hermes computer-use doctor` PASS(cua-driver 0.20.0, Accessibility/Screen Recording/AX healthy), `make build` PASS.
- `600/120/work60/break10`: fresh ready의 enabled Start와 disabled Skip, 고정 카드·통계·Esc 안내를 실제 화면/AX로 확인했고 기존 장시간 no-auto-start PASS를 재사용했다.
- `122/120`: 전체 `01:47`, short copy, Start/Skip에 AX press action 없음.
- 경계 fixture: `124/120` 초기 `02:04` enabled Start 확인 → 1.05초 뒤 Return → 다음 read-back이 전체 `02:02`이면서 running `01:59`; 122 guard와 함께 123초 경계 수락을 실제로 입증했다.
- 입력: Start click, Return, Space, Cancel→ready→restart, ready/running/completed Esc, skip-after 후 Skip이 모두 exit/state read-back으로 PASS했다.
- runtime: initial running AX는 `01:59`(첫 global tick 이후), bounded poll에서 exact `01:00`, 이어서 `완료했어요`/완료 안내를 관찰했다. completed epoch 1787205941.85904, 자동 exit epoch 1787205944.6247108로 약 2.766초 뒤 exit 0이었다.
- completed Esc: completed AX read-back 뒤 Escape를 한 번 보내 0.141초 내 exit 0.
- 접근성: ready Start의 지정 label/help, running title/time/instruction, Cancel label/help, completed title/value/help를 actual AX tree에서 확인했다. Tab/Shift-Tab key delivery와 이어진 Return 실행은 PASS했으나 Cua AX가 focused 속성을 노출하지 않아 focus ring/단계별 focused read-back은 N/A다. 실제 VoiceOver 음성과 announcement event count도 관찰 API가 없어 N/A이며, 소스/빌드의 announcement API만 유지 확인했다.
- 환경 N/A: Reduce Motion/Increase Contrast 설정을 건드리지 않았고, display가 1대라 multi-display를 실행하지 않았다.
- launchd: `cmd.Run()` blocking, 실행 중 StartInterval missed, helper 종료 뒤 다음 interval 재개라는 기존 source/자동 계약을 변경하지 않았다. LaunchAgent를 조작하는 실런타임 검증은 이번 승인 범위 밖이라 NOT RUN이다.
- 제품 결함: 없음. `02:00` 단일 프레임은 window discovery와 첫 1초 tick 경합으로 미포착했지만 `01:00`/120초 completed/약 3초 종료는 실제 연속 런타임으로 확인했다.

### 종료 품질 게이트

- `make test`: PASS — Go 전체 패키지 exit 0, Swift 101 tests/0 failures/0 unexpected.
- `git diff --check`: PASS — whitespace error 없음.
- `git diff --cached --name-only`: 출력 없음 — staged 파일 0개.
- 종료 `git status --short`의 보호 대상 목록은 시작 전 baseline과 동일하다: `.analysis/**`, `.serena/`, README, CHANGELOG, `BreakScreenApp/main.swift`, root PNG 및 기존 untracked guided source/test. 이번 QA가 허용 범위 밖 자산을 새로 수정·삭제·이동·스테이징하지 않았다.
- 이번 QA의 의도적 쓰기는 `docs/product/utility-mvp-manual-qa.md`, `utility-mvp-acceptance.md`, `utility-mvp-implementation.md`의 실제 결과 갱신뿐이다.
