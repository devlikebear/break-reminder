# Phase 1: 백엔드 pause 모드 3종 — 작업지시서

_이 페이즈의 목표를 달성하기 위한 구체적 작업 목록. Claude Code가 위에서부터 순차 실행하되, 체크포인트에서 사용자 확인을 받는다._

_작성일: 2026-05-12_
_속한 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)_
_예상 소요: 3-4시간_

## 페이즈 목표

`break-reminder pause --mode={meeting|focus|afk}` 와 `resume`이 CLI 레벨에서 완전히 동작한다. **모드별 업무시간 누적 정책이 다르게 적용된다**:

- **meeting**: 멈춘 동안 시간 anchor를 shift → `today_work_seconds` 누적 X. work_cycle 유지 (`WorkSeconds`는 멈춤 직전 값 그대로 보존).
- **focus**: anchor를 shift하지 않음 → 멈춘 시간이 그대로 work로 누적. resume 후 휴식 도달 시점이 짧아짐.
- **afk**: anchor shift + `WorkSeconds=0` 리셋. `today_work_seconds` 누적 X. 외출 후 깨끗하게 다시 시작.

기존 `pause/resume` (인자 없음) 호출은 기본 `meeting` 모드로 동작해 회귀 없음.

## 전제 조건

- [ ] 로드맵 확인 및 동의 완료
- [ ] `go test ./...`가 현재 상태에서 통과하는지 베이스라인 확인
- [ ] 기존 `internal/state/state.go` Pause/Resume 동작 숙지

## 포함 기능

1. `State.PauseReason` 필드 — `""|"meeting"|"focus"|"afk"`
2. `State.Pause(at int64, reason string)` 시그니처 변경 + 분기 로직
3. `State.Resume(at int64)` 분기 로직 (PauseReason 따라)
4. key=value 직렬화에 `PAUSE_REASON` 라인 추가 (Load/Save 양쪽)
5. CLI `pause --mode=` 플래그 추가 (cobra)
6. 단위 테스트 — 3가지 모드 × accrual 시나리오

## 이 페이즈에서 하지 않는 것

- GUI 변경 → Phase 2
- Auto-resume 타이머 (`PauseUntil`) → Phase 3
- HelperCore Swift 파서의 `PAUSE_REASON` 반영 → Phase 2 (GUI에서 필요해질 때)
- 설정 관련 작업 → Phase 4/5

## 작업 체크리스트

### 작업 그룹 A: state schema 확장

- [ ] **T1.A.1** — `State` 구조체에 `PauseReason` 필드 추가
  - 파일: `internal/state/state.go`
  - 내용:
    - `State` struct에 `PauseReason string \`json:"pause_reason"\`` 추가 (Paused/PausedAt 바로 아래)
    - 상수 정의 (파일 상단 또는 별도 const 블록):
      ```go
      const (
          PauseReasonMeeting = "meeting"
          PauseReasonFocus   = "focus"
          PauseReasonAFK     = "afk"
      )
      ```
  - 참조: 기존 필드 네이밍 컨벤션 (snake_case json tag)
  - 검증: `go build ./internal/state/...` 통과

- [ ] **T1.A.2** — `serialize`/`Load`에 `PAUSE_REASON` 추가
  - 파일: `internal/state/state.go`
  - 내용:
    - `serialize(s State)`에 `fmt.Fprintf(&b, "PAUSE_REASON=%s\n", s.PauseReason)` 추가 (PAUSED_AT 다음 줄)
    - `Load`의 switch에 `case "PAUSE_REASON":` 분기. 유효값(`meeting/focus/afk` 또는 `""`)만 받음. 빈 값/잘못된 값은 `""`으로.
  - 참조: 기존 `Load` 함수의 다른 case들과 동일 스타일
  - 검증: 단위 테스트 추가 (T1.C.1에서)

### 작업 그룹 B: Pause/Resume 분기 로직

- [ ] **T1.B.1** — `Pause` 시그니처 변경 + 모드 분기
  - 파일: `internal/state/state.go`
  - 내용:
    - 기존 `func (s State) Pause(at int64) State`를 `func (s State) Pause(at int64, reason string) State`로 변경
    - `reason`이 유효한 값이 아니면 (`meeting/focus/afk` 외) 기본값 `meeting` 사용
    - 기존 로직 (`accrueUntil`, `Paused=true`, `PausedAt=at`)은 그대로
    - `s.PauseReason = reason` 추가
    - `focus` 모드의 핵심 특수 처리: **`PausedAt`을 0으로 둠** — Resume에서 anchor shift 안 하도록 신호로 사용. (또는 별도 플래그를 두는 대신 PauseReason으로 판단)
      - 권장: `PausedAt`은 항상 기록하고, **Resume에서 `PauseReason == focus`면 shift를 스킵**하는 방식이 더 명확.
  - 검증: `state_test.go`의 기존 Pause 테스트가 호출부 수정만으로 통과해야 함

- [ ] **T1.B.2** — `Resume` 모드별 분기
  - 파일: `internal/state/state.go`
  - 내용:
    - 함수 시그니처는 그대로 `func (s State) Resume(at int64) State`
    - 진입 시 `reason := s.PauseReason` 저장
    - 분기:
      - `reason == PauseReasonFocus`: anchor shift 스킵. 즉 기존 로직의 `s.LastCheck += gap`, `s.BreakStart += gap`, `s.SnoozeUntil += gap` 부분을 실행하지 않음. 대신 `accrueUntil(at)` 호출해서 멈춘 시간을 work로 누적.
      - `reason == PauseReasonAFK`: 기존 anchor shift 로직 그대로 + 종료 시 `s.WorkSeconds = 0`, `s.SnoozeUntil = 0`. work cycle 리셋.
      - `reason == PauseReasonMeeting` 또는 그 외: 기존 anchor shift 로직 그대로 (현행 유지).
    - 함수 끝에서 `s.PauseReason = ""` 초기화
  - 의존: T1.B.1 완료 후
  - 검증: T1.C.2 단위 테스트

- [ ] **T1.B.3** — 기존 호출부 갱신
  - 파일: 다음을 grep하여 모두 갱신
    - `cmd/break-reminder/pause_resume.go`
    - `cmd/break-reminder/break.go` 등 `s.Pause(` 호출하는 곳
    - 기타 `internal/` 내부에서 호출되는 곳 (없으면 스킵)
  - 내용:
    - `s.Pause(at)` → `s.Pause(at, state.PauseReasonMeeting)` 또는 적절한 모드로
    - cobra 명령 자체는 T1.B.4에서 `--mode` 플래그 추가하므로, 일단 임시로 `PauseReasonMeeting` 기본값 사용
  - 검증: `go build ./...` 통과

- [ ] **T1.B.4** — CLI `pause --mode=` 플래그
  - 파일: `cmd/break-reminder/pause_resume.go`
  - 내용:
    - `newPauseCmd()`에 cobra Flag 추가: `var modeFlag string`, `cmd.Flags().StringVar(&modeFlag, "mode", "meeting", "Pause mode: meeting|focus|afk")`
    - `RunE` 진입 시 mode 검증:
      ```go
      validModes := map[string]bool{
          state.PauseReasonMeeting: true,
          state.PauseReasonFocus:   true,
          state.PauseReasonAFK:     true,
      }
      if !validModes[modeFlag] {
          return fmt.Errorf("invalid --mode %q (must be meeting|focus|afk)", modeFlag)
      }
      ```
    - `s.Pause(nowFunc().Unix())` → `s.Pause(nowFunc().Unix(), modeFlag)`
    - 성공 메시지에 모드 포함: `fmt.Fprintf(cmd.OutOrStdout(), "Timer paused (%s mode, reason=%s).\n", pausedMode, modeFlag)`
    - 로그 메시지에도 모드 포함
  - 참조: 기존 `--mode` 같은 cobra 플래그 사용 패턴 — `cmd/break-reminder/snooze.go`에 `--duration` 플래그가 있다면 그 패턴 참조
  - 검증: `break-reminder pause --mode=focus` / `--mode=invalid`로 수동 실행해서 동작/에러 확인

### 작업 그룹 C: 테스트

- [ ] **T1.C.1** — Load/Save round-trip에 `PauseReason` 포함
  - 파일: `internal/state/state_test.go`
  - 내용:
    - 기존 `TestSaveLoadRoundTrip` 또는 유사 테스트에 `s.PauseReason = "focus"` 케이스 추가
    - 잘못된 reason 값(`"invalid"`)을 파일에 직접 쓴 뒤 Load 시 `""`로 정규화되는지 확인하는 테스트
  - 검증: `go test ./internal/state/...` 통과

- [ ] **T1.C.2** — 3가지 모드 × accrual 시나리오
  - 파일: `internal/state/state_test.go`
  - 내용 (table-driven 권장):
    ```
    시나리오 A — meeting:
      초기: work mode, WorkSeconds=600, LastCheck=t0, TodayWorkSeconds=600
      Pause(t0, "meeting")
      ... 300초 경과 ...
      Resume(t0+300)
      기대: WorkSeconds=600, TodayWorkSeconds=600, LastCheck=t0+300, PauseReason=""

    시나리오 B — focus:
      초기: 같음
      Pause(t0, "focus")
      ... 300초 경과 ...
      Resume(t0+300)
      기대: WorkSeconds=900, TodayWorkSeconds=900, LastCheck=t0+300, PauseReason=""

    시나리오 C — afk:
      초기: 같음
      Pause(t0, "afk")
      ... 300초 경과 ...
      Resume(t0+300)
      기대: WorkSeconds=0, TodayWorkSeconds=600, LastCheck=t0+300, SnoozeUntil=0, PauseReason=""
    ```
  - 참조: 기존 `state_test.go`의 table-driven 패턴
  - 검증: `go test ./internal/state/... -run TestPauseResume -v` 통과

- [ ] **T1.C.3** — CLI 통합 테스트
  - 파일: `cmd/break-reminder/pause_resume_test.go`
  - 내용:
    - 기존 pause_resume_test.go가 있다면 거기에 `--mode` 케이스 추가
    - `--mode=focus` 후 state 파일 읽어서 `PAUSE_REASON=focus` 확인
    - `--mode=invalid` 시 에러 반환 확인
  - 검증: `go test ./cmd/break-reminder/... -run TestPause -v` 통과

---

## ✅ Phase 1 Checkpoint

이 페이즈가 완료되었다고 판단하기 위한 검증:

**구현 확인:**
- [ ] 모든 작업 체크박스 완료
- [ ] `State.PauseReason` 필드가 추가되고 Load/Save에 반영됨
- [ ] `Pause(at, reason)` 시그니처가 변경되고 모든 호출부가 갱신됨
- [ ] `Resume`이 PauseReason에 따라 3가지 분기를 보여줌

**자동 검증:**
- [ ] 전체 테스트: `go test ./...` 통과
- [ ] 주요 패키지 테스트: `go test ./internal/state/... ./cmd/break-reminder/... -v` 통과
- [ ] 빌드: `go build ./...` 무경고

**수동 확인:**
- [ ] meeting 시나리오: `break-reminder pause --mode=meeting` → 1분 대기 → `resume` → `break-reminder status`에서 `today_work_seconds`가 거의 안 변함
- [ ] focus 시나리오: `break-reminder pause --mode=focus` → 1분 대기 → `resume` → `today_work_seconds`가 약 60초 더 증가 (※ daemon이 멈춰있어서 직접 검증 어려우면, 테스트로 대체 가능. 그래도 한 번은 수동 실행)
- [ ] afk 시나리오: `break-reminder pause --mode=afk` → 1분 대기 → `resume` → state 파일에서 `WORK_SECONDS=0`
- [ ] 잘못된 모드: `break-reminder pause --mode=invalid`가 에러 반환

**완료 처리:**
1. 위 항목 모두 통과 시, Claude Code는 사용자에게 다음을 보고:
   - 완료된 작업 요약 (T1.A.1 ~ T1.C.3)
   - 자동 검증 결과 (`go test ./...` 출력)
   - 수동 확인 시나리오의 실제 결과
2. 사용자가 명시적 승인 ("Phase 1 완료, 다음 진행") 후 Phase 2로 이동.
3. 체크포인트 실패 시: 실패 항목 보고 → 원인 분석 → 수정 → 재검증.

---

## 참고 자료

- 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)
- 기존 Pause/Resume 구현: `internal/state/state.go:66-161`
- 기존 CLI 패턴: `cmd/break-reminder/pause_resume.go`, `snooze.go`
- 테스트 컨벤션: `internal/state/state_test.go` (table-driven)
- CLAUDE.md "Key Review Checklist" — 상태 파일 새 필드 추가 시 Load/Save 양쪽 반영 강조

## 메모 / 주의

- **CLAUDE.md 리뷰 체크리스트 명시 사항**: "상태 파일: `state.Save()`에서 새 필드 추가 시 Load/Save 양쪽 반영 확인" — T1.A.2가 정확히 이 포인트.
- **focus 모드의 의미**: "이 멈춤도 업무 시간으로 친다"는 건, 풀어 말하면 "anchor를 안 옮긴다" = "Tick이 다음에 깨어났을 때 paused 동안의 elapsed가 그대로 누적된다". 단, 이 페이즈에서 `state.Resume`이 `accrueUntil(at)`을 명시적으로 호출하도록 했음 — Resume 시점에 한 번 정산하면 그 이후 Tick은 정상 동작. 한쪽이 누락되면 안 되므로 T1.C.2 테스트가 핵심.
- **focus 모드 + 휴식 트리거**: focus 모드로 30분 멈춰서 `WorkSeconds`가 `WorkDurationSec`을 초과하면 Resume 직후 Tick에서 휴식이 즉시 발동될 수 있음. 의도된 동작이므로 그대로 둔다 (사용자 입장: "집중 끝났으니 휴식하라"는 자연스러운 흐름).
- **기존 회귀**: 인자 없는 `break-reminder pause`는 cobra 플래그 기본값 `"meeting"`으로 자동 적용 → 기존 사용자 워크플로우 그대로 유지.

---
_다음 페이즈: Phase 2 — GUI 멈춤 컨트롤 → [`pause-and-settings-phase-2-gui-pause-control.md`](./pause-and-settings-phase-2-gui-pause-control.md)_
