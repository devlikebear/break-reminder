# GUI 강제 멈춤 + 설정 관리 — 개발 로드맵

_이 문서는 두 가지 신규 기능(강제 멈춤 / 설정 관리 UI)을 GUI 대시보드에 통합하는 전체 계획의 총괄본. Claude Code는 이 로드맵을 먼저 읽고 전체 맥락을 파악한 뒤, 각 페이즈 작업지시서로 이동해 구현을 진행한다. 페이즈 간 전환 시 반드시 사용자 확인을 받는다._

_작성일: 2026-05-12_
_예상 전체 소요: 14-18시간 (≈ 1주, 사이드 프로젝트 페이스)_
_페이즈 수: 5개_

## Overview

`break-reminder`는 50분 작업 / 10분 휴식 사이클을 강제하는 macOS 도구. 현재 GUI 대시보드(`helpers/Sources/DashboardApp`)는 타이머 상태와 통계를 보여주지만, **(a) 회의·외출 같은 상황에서 타이머를 일시정지하는 UI가 없고**, **(b) 설정 변경은 `~/.config/break-reminder/config.yaml`을 직접 편집해야 한다.**

이번 작업으로 두 가지를 추가한다:

1. **강제 멈춤 기능 — 모드별 정책**: 회의/집중/외출 3가지 모드 중 선택해서 타이머를 멈출 수 있다. 모드별로 "이 시간을 업무로 칠지" 정책이 다르다 (회의: 누적 X, 집중: 누적 O, 외출: 리셋). 시간을 지정하면 자동으로 해제된다.
2. **설정 관리 UI**: 대시보드에 "설정" 탭을 추가해서 핵심 8-10개 필드(작업/휴식 시간, idle 임계값, 근무 스케줄, 알림/TTS/break-screen 토글 등)를 폼으로 편집한다. 저장은 CLI `break-reminder config set`을 호출해서 검증(`validateSchedule`)을 보장한다.

기존 `state.Pause()/Resume()` 헬퍼와 `config.Save()/ApplyYAMLChanges()`가 이미 견고하게 구현되어 있어, 이번 작업은 그 위에 "모드 개념"과 "GUI 진입점"을 얹는 성격이다.

## 완료 조건 (전체)

모든 페이즈 완료 시 달성할 상태:

- [ ] CLI에서 `break-reminder pause --mode={meeting|focus|afk} [--duration=30m]` 동작
- [ ] CLI에서 `break-reminder config set work_duration_min=45` 같은 명령으로 설정 변경 가능 (검증 포함)
- [ ] GUI 대시보드 헤더에 멈춤 모드 picker + 자동 해제 시간 선택 + Resume 버튼 노출
- [ ] GUI에 "설정" 탭이 추가되고, 핵심 필드 폼으로 편집/저장 가능
- [ ] 멈춤 모드별 업무시간 누적 정책이 실제로 다르게 동작 (회의: 누적 X / 집중: 누적 O / 외출: 리셋)
- [ ] Auto-resume 시간이 경과하면 다음 tick에서 자동 해제
- [ ] 모든 신규 기능에 단위 테스트가 추가되고 `go test ./...` 통과
- [ ] 기존 기능(snooze, force break, reset)이 회귀 없이 동작
- [ ] CHANGELOG.md + README.md 업데이트

## 기술 스택 / 환경

- **언어 / 런타임**: Go 1.22+ (백엔드/CLI), Swift 5.x / SwiftUI (GUI)
- **주요 패키지**:
  - Go: `cobra` (CLI), `gopkg.in/yaml.v3`, 표준 testing
  - Swift: SwiftPM, `HelperCore` 공유 라이브러리
- **저장소**: 평문 파일 (`~/.break-reminder-state` key=value, `~/.config/break-reminder/config.yaml`)
- **실행 환경**: macOS 12+, launchd로 데몬 실행 (60초 check interval)

참고:
- `.analysis/AI_CONTEXT.md` — 모듈 분석 결과
- `CLAUDE.md` — 워크플로우, 리뷰 체크리스트
- `config/default.yaml` — 설정 키 전체 목록과 기본값

## Out of Scope

명시적으로 이번 범위에서 뺀다 (Claude Code가 "이것도 해야 하나?" 고민 방지):

- **캘린더 연동 자동 멈춤**: 회의 시간 자동 감지. 향후 별도 페이즈.
- **TTS API key / AI CLI 같은 고급 설정의 GUI 편집**: 보안 민감값 + 자주 안 바뀜 → yaml 직접 편집 유지.
- **Bubbletea TUI 대시보드(`internal/dashboard`)의 pause UI 추가**: 이번엔 SwiftUI GUI만. TUI는 다음 기회.
- **멈춤 이력 통계화**: "오늘 회의로 멈춘 시간" 같은 지표는 인사이트 탭에 미반영.
- **launchd 데몬 실행 / `bin/install` 흐름 변경**: 기존 설치 방식 그대로 유지.

## 페이즈 구성

각 페이즈는 **수직 슬라이스** — 끝나면 동작하는 무언가가 나오는 단위.

### Phase 1: 백엔드 pause 모드 3종 — Meeting/Focus/AFK 정책 분기

- **목표**: CLI로 `break-reminder pause --mode=meeting|focus|afk` 와 `resume`이 정확히 동작. 모드별 업무시간 누적 정책이 다르게 적용됨. 기존 `pause/resume` 회귀 없음.
- **포함 기능**: state schema 확장 (`PauseReason`), `Pause(at, reason)` / `Resume(at)` 분기, CLI 플래그
- **예상 소요**: 3-4시간
- **작업지시서**: [`pause-and-settings-phase-1-backend-pause-modes.md`](./pause-and-settings-phase-1-backend-pause-modes.md)
- **Checkpoint 요약**: `go test ./...` 통과 + CLI 3가지 모드 수동 시나리오 동작

### Phase 2: GUI 멈춤 컨트롤 — 헤더에 모드 picker

- **목표**: 대시보드를 켠 상태에서 GUI 클릭만으로 멈춤/재개 가능. 모드(회의/집중/외출) 선택 가능. 상태 표시도 모드별로 다름.
- **포함 기능**: `AppState.pauseReason` 필드, `StatusHeaderView`에 모드 picker + Pause/Resume 버튼, CLI Process 호출
- **예상 소요**: 3-4시간
- **작업지시서**: [`pause-and-settings-phase-2-gui-pause-control.md`](./pause-and-settings-phase-2-gui-pause-control.md)
- **Checkpoint 요약**: GUI에서 3가지 모드로 멈춤/재개 동작, state 파일에 `PAUSE_REASON` 반영

### Phase 3: Auto-resume 타이머 — 시간 지정 자동 해제

- **목표**: "30분 후 자동 해제" 시나리오. 회의 끝나고 깜빡해도 알아서 재개. GUI에 남은 시간 카운트다운 표시.
- **포함 기능**: `state.PauseUntil` 필드, `timer.Tick`에서 자동 해제 분기, GUI에 duration picker(15/30/60분/직접입력) + 카운트다운
- **예상 소요**: 2-3시간
- **작업지시서**: [`pause-and-settings-phase-3-auto-resume.md`](./pause-and-settings-phase-3-auto-resume.md)
- **Checkpoint 요약**: `PauseUntil` 도달 시 다음 tick에서 자동 Resume, GUI 카운트다운 정확

### Phase 4: config set CLI + HelperCore 확장

- **목표**: CLI로 모든 설정 키를 변경 가능 (`config set k=v`). 검증/원자성 보장. Swift HelperCore가 전체 config 필드를 읽고 직렬화 가능 (단, 저장은 CLI에 위임).
- **포함 기능**: `break-reminder config set` 명령 (cobra), `ConfigParser` 확장 (모든 필드 + work_days 배열), CLI 호출 유틸 in Swift
- **예상 소요**: 2시간
- **작업지시서**: [`pause-and-settings-phase-4-config-set-cli.md`](./pause-and-settings-phase-4-config-set-cli.md)
- **Checkpoint 요약**: CLI로 `work_duration_min=45` 설정 변경 후 `config show`에 반영, 잘못된 값은 거부됨

### Phase 5: Settings UI 탭 — 핵심 필드 폼

- **목표**: 대시보드에 "설정" 탭이 추가되고, 핵심 8-10개 필드를 폼으로 편집할 수 있다. 저장 버튼 클릭 시 CLI 호출로 yaml에 반영되고 검증 실패 시 에러 표시.
- **포함 기능**: `SettingsTabView`, `TabBarView`에 탭 추가, 폼 상태 관리, 저장/취소/기본값 복원, 에러 표시
- **예상 소요**: 4-5시간
- **작업지시서**: [`pause-and-settings-phase-5-settings-ui.md`](./pause-and-settings-phase-5-settings-ui.md)
- **Checkpoint 요약**: GUI에서 work_duration_min 변경 → yaml 반영 → 다음 break까지의 시간에 영향

## 페이즈 간 의존성

```
Phase 1 (백엔드 pause 모드)
  └─→ Phase 2 (GUI 멈춤 컨트롤)
        ├─→ Phase 3 (Auto-resume 타이머)
        └─→ Phase 4 (config set CLI)
              └─→ Phase 5 (Settings UI 탭)
```

**병렬 가능성**: Phase 3과 Phase 4는 Phase 2 완료 후 서로 독립적으로 진행 가능. 단, 사이드 프로젝트 1인 개발이므로 보통 순차 진행이 더 단순.

## 페이즈 간 전환 규칙

각 페이즈는 다음 조건을 만족해야 완료로 간주:

1. 작업지시서의 모든 체크박스 완료
2. Checkpoint 블록의 모든 검증 통과
3. 사용자가 명시적으로 "Phase N 완료 확인, 다음 진행" 승인

Checkpoint가 실패하면:
- Claude Code가 실패 원인을 보고
- 사용자와 함께 원인 파악
- 수정 후 재검증
- 재검증 통과 후 사용자 승인 → 다음 페이즈

## 최종 완료 체크리스트 (전 페이즈 종료 후)

- [ ] 모든 페이즈 Checkpoint 통과
- [ ] 전체 테스트: `go test ./...` 통과
- [ ] Swift 빌드: `cd helpers && swift build` 통과
- [ ] 핵심 시나리오 1 — "회의 멈춤 → 30분 후 자동 재개": GUI로 meeting 모드 + 30분 선택, 30분 후 timer가 정확히 work 모드로 복귀, 그 30분이 `today_work_seconds`에 더해지지 않음
- [ ] 핵심 시나리오 2 — "GUI로 설정 변경": Settings 탭에서 work_duration_min 50 → 45 변경 후 저장, `cat ~/.config/break-reminder/config.yaml`에서 확인, 다음 break 진입 시점이 45분 기준으로 동작
- [ ] 핵심 시나리오 3 — "Focus 모드 멈춤": focus 모드로 1분 멈춤 후 resume, `today_work_seconds`에 그 1분이 누적되어 있음
- [ ] Out of Scope 항목 미구현 확인 (캘린더, TUI pause, 멈춤 통계 등)
- [ ] 기존 컨벤션 준수 — `internal/` 모듈 경계, snake_case yaml 키, key=value state 형식, godoc 영어 주석
- [ ] CHANGELOG.md에 "Added: GUI pause modes / settings panel" 섹션 추가
- [ ] README.md 업데이트 — 새 기능 사용법 한 단락
- [ ] (선택) VERSION 파일 minor bump 후 릴리스 (CLAUDE.md의 Release 절차 따름)

## 참고 자료

- 코드베이스 분석: `.analysis/AI_CONTEXT.md`, `.analysis/outputs/`
- 개발 워크플로우: `CLAUDE.md` (PR review, release 절차, 리뷰 체크리스트)
- 기존 pause 구현: `internal/state/state.go` (Pause/Resume 헬퍼), `cmd/break-reminder/pause_resume.go`
- 기존 config 저장: `internal/config/load.go` (`Save`, `ApplyYAMLChanges`, `validateSchedule`)
- 기존 GUI 진입점: `helpers/Sources/DashboardApp/DashboardAppMain.swift`, `DashboardViewModel.swift`
- GUI↔백엔드 통신 패턴 (CLI 호출): `DashboardViewModel.runInsightsRefresh()`
- GUI↔백엔드 통신 패턴 (state 직접 write): `DashboardViewModel.forceBreak()`, `SystemIO.writeStateToDisk()`

---
_Claude Code 사용 시: 이 로드맵을 먼저 읽고 전체 맥락 파악. 그다음 Phase 1 작업지시서로 이동해서 구현 시작. 페이즈 전환은 반드시 사용자 명시적 승인 후._
