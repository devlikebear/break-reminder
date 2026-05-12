# Phase 3: Auto-resume 타이머 — 작업지시서

_작성일: 2026-05-12_
_속한 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)_
_예상 소요: 2-3시간_

## 페이즈 목표

사용자가 멈춤을 시작할 때 "30분 후 자동 해제" 같은 시간 한도를 지정할 수 있다. 시간이 지나면 다음 timer tick(60초 주기)에서 자동으로 Resume이 트리거된다. GUI 헤더에는 남은 시간 카운트다운이 표시된다.

## 전제 조건

- [ ] Phase 2 완료 및 사용자 승인
- [ ] GUI 멈춤/재개가 안정적으로 동작
- [ ] daemon이 launchd로 실행 중 (또는 `break-reminder check` 수동 실행 가능 환경)

## 포함 기능

1. `State.PauseUntil` 필드 (`int64`, 0이면 자동 해제 비활성)
2. `State.Pause(at, reason, duration)` 시그니처 확장 (또는 별도 헬퍼)
3. `timer.Tick`에서 paused 상태 처리 시: `PauseUntil > 0 && now >= PauseUntil`이면 자동 Resume 트리거
4. CLI: `break-reminder pause --duration=30m` 플래그 추가
5. GUI: 멈춤 메뉴 진입 시 duration picker (15분/30분/1시간/직접입력/무제한)
6. GUI: paused 상태 헤더에 남은 시간 카운트다운 (예: "PAUSED · 회의 · 28:42 남음")

## 이 페이즈에서 하지 않는 것

- 자동 해제 시점 알림(notification) → 현재는 다음 tick에서 무음 해제. notify 추가는 추후 고려.
- 일시정지 이력 통계 → Out of Scope
- Settings 탭 → Phase 5

## 작업 체크리스트

### 작업 그룹 A: state 확장

- [ ] **T3.A.1** — `PauseUntil` 필드 추가
  - 파일: `internal/state/state.go`
  - 내용:
    - `State` struct에 `PauseUntil int64 \`json:"pause_until"\`` 추가
    - `serialize`에 `PAUSE_UNTIL=` 라인 추가
    - `Load` switch에 `case "PAUSE_UNTIL":` 분기 (int64 파싱)
  - 검증: `go build ./internal/state/...` 통과

- [ ] **T3.A.2** — `Pause` 시그니처 확장
  - 파일: `internal/state/state.go`
  - 내용:
    - 옵션 A (권장): `Pause(at int64, reason string, durationSec int) State`. `durationSec <= 0`이면 `PauseUntil = 0` (무제한).
    - 옵션 B: 기존 시그니처 유지하고 별도 `SetPauseUntil(until int64) State` 헬퍼.
    - 권장 옵션 A. 모든 호출부 갱신 (Phase 1과 같은 패턴).
    - Pause 진입 시: `if durationSec > 0 { s.PauseUntil = at + int64(durationSec) } else { s.PauseUntil = 0 }`
    - Resume에서: `s.PauseUntil = 0` 초기화
  - 의존: T3.A.1
  - 검증: 단위 테스트 (T3.D.1에서)

### 작업 그룹 B: Tick 자동 해제

- [ ] **T3.B.1** — `timer.Tick`에서 자동 Resume
  - 파일: `internal/timer/timer.go`
  - 내용:
    - 현재 `if s.Paused { return result }` 라인을 다음과 같이 변경:
      ```go
      if s.Paused {
          if s.PauseUntil > 0 && unix >= s.PauseUntil {
              // Auto-resume
              result.State = result.State.Resume(unix)
              result.LogMsg = "Auto-resumed from " + s.PauseReason + " pause"
              // Tick 계속 진행 (이번 사이클에 work/break 로직도 한 번 돌게)
              // 단, LastCheck가 Resume에서 이미 갱신되었거나 anchor shift된 상태이므로
              // 아래 'elapsed := int(unix - s.LastCheck)' 계산이 0이 됨.
              // → switch s.Mode 분기 진입은 가능. work 모드라면 idle만 체크하고 가벼운 패스.
          } else {
              return result
          }
      }
      ```
    - 주의: Resume이 anchor를 shift하므로 (`focus` 모드 제외) 자동 해제 후 `elapsed`가 거의 0이 되어 work/break 누적은 미미. 이게 의도된 동작.
    - `focus` 모드 자동 해제 시: 멈춘 시간이 work로 누적 → 즉시 break 트리거 가능. 그대로 둠 (자연스러운 흐름).
  - 의존: T3.A.2
  - 검증: T3.D.2 단위 테스트

### 작업 그룹 C: CLI 플래그

- [ ] **T3.C.1** — `pause --duration=` 플래그
  - 파일: `cmd/break-reminder/pause_resume.go`
  - 내용:
    - cobra Flag 추가: `var durationFlag string`, `cmd.Flags().StringVar(&durationFlag, "duration", "", "Auto-resume after duration (e.g., 30m, 1h). Empty = no auto-resume")`
    - `RunE`에서 파싱:
      ```go
      var durationSec int
      if durationFlag != "" {
          d, err := time.ParseDuration(durationFlag)
          if err != nil {
              return fmt.Errorf("invalid --duration %q: %w", durationFlag, err)
          }
          if d <= 0 {
              return fmt.Errorf("--duration must be positive")
          }
          durationSec = int(d.Seconds())
      }
      ```
    - `s.Pause(now, modeFlag, durationSec)` 호출
    - 성공 메시지에 자동 해제 시각 포함: `if durationSec > 0 { fmt.Fprintf(..., " (auto-resume at %s)", time.Unix(now+int64(durationSec), 0).Format("15:04")) }`
  - 참조: 기존 `snooze.go`의 duration 파싱 패턴 (있으면 그대로)
  - 검증: `break-reminder pause --mode=meeting --duration=1m` 수동 실행

### 작업 그룹 D: 테스트

- [ ] **T3.D.1** — state 단위 테스트
  - 파일: `internal/state/state_test.go`
  - 내용:
    - `Pause(t0, "meeting", 1800)` 후 `s.PauseUntil == t0 + 1800` 확인
    - `Pause(t0, "meeting", 0)` 후 `s.PauseUntil == 0` 확인
    - Resume 후 `s.PauseUntil == 0` 확인
  - 검증: `go test ./internal/state/... -v` 통과

- [ ] **T3.D.2** — timer 자동 해제 테스트
  - 파일: `internal/timer/timer_test.go`
  - 내용:
    - 시나리오: `s.Paused = true, s.PauseReason = "meeting", s.PauseUntil = t0 + 1800`
    - `Tick(cfg, s, t0 + 1799, idle=0)` → state 여전히 paused
    - `Tick(cfg, s, t0 + 1801, idle=0)` → state.Paused == false, PauseReason == "", PauseUntil == 0
  - 검증: `go test ./internal/timer/... -v` 통과

### 작업 그룹 E: GUI duration picker

- [ ] **T3.E.1** — `pause(mode:duration:)` ViewModel 메서드 확장
  - 파일: `helpers/Sources/DashboardApp/DashboardViewModel.swift`
  - 내용:
    - 기존 `pause(mode:)` 시그니처를 `pause(mode: String, durationMinutes: Int?)` 로 변경
    - CLI 호출 시 duration이 있으면 `["pause", "--mode=\(mode)", "--duration=\(durationMinutes!)m"]`
    - 없으면 기존대로 `["pause", "--mode=\(mode)"]`
  - 검증: 컴파일 통과

- [ ] **T3.E.2** — duration picker UI
  - 파일: `helpers/Sources/DashboardApp/StatusHeaderView.swift`
  - 내용:
    - 기존 Menu 구조를 2단계로 변경:
      - 1단계: 모드 선택 (회의/집중/외출)
      - 2단계: duration 선택 — Menu 내부에 sub-menu 또는 모드 선택 후 `@State` 변수 띄우고 sheet/popover로 picker 표시
    - 빠른 옵션: `15분`, `30분`, `1시간`, `2시간`, `무제한`. 옵션은 SwiftUI Picker 또는 Menu 항목.
    - 단순화 경로 (권장): Menu에서 모드 × duration 조합을 평면 나열
      - "회의 30분", "회의 1시간", "회의 무제한", "집중 30분", ... (총 9-12개 항목)
      - 또는 hierarchical Menu — 최상위 모드, sub-menu duration
  - 참조: SwiftUI `Menu { ... }` 내부에 `Menu("회의") { Button("30분", action: ...) }` 형태로 중첩 가능
  - 검증: 빌드 + 수동 — 메뉴 펼쳤을 때 옵션이 합리적으로 보이는지

- [ ] **T3.E.3** — 카운트다운 표시
  - 파일: `helpers/Sources/DashboardApp/DashboardViewModel.swift`, `StatusHeaderView.swift`
  - 내용:
    - ViewModel: `pauseRemainingText: String` 계산 속성
      ```swift
      var pauseRemainingText: String? {
          guard isPaused, state.pauseUntil > 0 else { return nil }
          let remaining = max(0, state.pauseUntil - now)
          let m = remaining / 60
          let s = remaining % 60
          return String(format: "%d:%02d 남음", m, s)
      }
      ```
    - `AppState.pauseUntil` 필드 추가 (Phase 2의 T2.A.1 패턴 따라 StateParser/Serializer 확장)
    - 헤더에서 paused 상태일 때 statusText 뒤에 `pauseRemainingText` 출력
    - 기존 1초마다 refresh()가 도는 타이머가 이미 있어서 카운트다운은 자동으로 갱신됨
  - 의존: T3.A.1 (PauseUntil 직렬화)
  - 검증: GUI에서 `pause --duration=2m` 후 카운트다운이 1초마다 줄어드는지

### 작업 그룹 F: 통합 시나리오

- [ ] **T3.F.1** — End-to-end 시나리오 검증
  - 파일: 없음 (수동)
  - 내용:
    1. daemon 실행 중 확인 (`launchctl list com.devlikebear.break-reminder`)
    2. GUI에서 "회의 1분" 선택
    3. 60-120초 대기 (check interval 고려)
    4. GUI 헤더가 자동으로 "WORKING"으로 돌아오는지 확인
    5. state 파일에서 `PAUSED=false`, `PAUSE_UNTIL=0`, `PAUSE_REASON=` 확인
  - 검증: 위 시나리오 정상 동작

---

## ✅ Phase 3 Checkpoint

**구현 확인:**
- [ ] 모든 작업 체크박스 완료
- [ ] `PauseUntil` 필드 + 직렬화 완료
- [ ] CLI `--duration` 플래그 + GUI duration picker 동작
- [ ] timer tick에서 자동 Resume 트리거

**자동 검증:**
- [ ] `go test ./...` 통과
- [ ] `cd helpers && swift build` 통과

**수동 확인:**
- [ ] 1분 자동 해제 E2E 시나리오 (T3.F.1) 성공
- [ ] 카운트다운이 GUI에 1초마다 갱신
- [ ] focus 모드 1분 자동 해제 → resume 후 work_seconds가 60초 증가 확인 (또는 break 발동 시점이 1분 앞당겨짐)
- [ ] duration 없이 (무제한) 멈춤도 여전히 동작 — `PauseUntil=0`이면 자동 해제 안 일어남

**완료 처리:**
1. 사용자에게 완료 보고
2. 사용자 승인 후 Phase 4로 이동.
3. 실패 시: 원인 분석 → 수정 → 재검증.

---

## 참고 자료

- 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)
- 기존 snooze duration 파싱: `cmd/break-reminder/snooze.go`
- 기존 timer tick 구조: `internal/timer/timer.go:41-104`
- CLAUDE.md "Key Review Checklist" — "timer 로직: 기본 CheckIntervalSec(60초)과 launchd(60초) 기준으로 실제 도달 가능한 타이밍인지" — T3.F.1 시나리오를 1분으로 잡은 이유.

## 메모 / 주의

- **체크 주기**: launchd가 60초마다 데몬을 깨우므로, `--duration=1m`이면 정확히 1분 후가 아니라 최대 1분 ~ 1분 59초 사이에 해제. 사용자에게는 합리적 정확도.
- **자동 해제 시 알림**: 현재 무음으로 해제. 사용자가 "회의 끝났는데 알림이 안 와서 멈춤이 안 풀린 줄 알았다"면 추후 `ActionNotifyAutoResume` 추가 검토. 이번엔 의도적으로 단순화.
- **focus 모드 + 자동 해제**: focus 모드로 30분 멈춰서 work_seconds가 work_duration_sec을 초과한 상태에서 자동 해제되면 즉시 break 트리거. 의도된 흐름.
- **시계 변경 / 시스템 슬립**: timer.Tick은 이미 elapsed > 3600초나 maxExpectedElapsed 초과 시 리셋하는 가드가 있음. PauseUntil 로직과 충돌하지 않는지 확인 — 슬립 후 깨어났을 때 elapsed가 maxExpectedElapsed를 넘으면 LastCheck만 갱신하고 return. 이 경로에서도 자동 Resume이 동작하도록, `if s.Paused { ... }` 블록이 elapsed 가드보다 앞에 있는 현재 구조 유지.

---
_다음 페이즈: Phase 4 — config set CLI + HelperCore 확장 → [`pause-and-settings-phase-4-config-set-cli.md`](./pause-and-settings-phase-4-config-set-cli.md)_
