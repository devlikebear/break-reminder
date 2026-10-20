# Phase 1: 업무 맥락 — 작업지시서

_작성일: 2026-10-20 (rev.2) · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 24–36h_

## 페이즈 목표

일정과 앱 사용을 기존 업무 리듬과 결합한다. 끝나면 팝오버에 다음 일정과 "다음 회의까지 집중 가능 시간"이 보이고, 원클릭으로 그 시간에 맞춘 뽀모도로를 시작하며, 회의 중에는 휴식 화면이 뜨지 않는다.

## 전제 조건

- [ ] P0 완료, 스파이크에서 EventKit 권한 유지 PASS

## 포함 기능

1. 캘린더 맥락 — 오늘/다음 일정 (Swift 수집 → 엔진 판단)
2. 회의 인식 휴식 — 회의 중 자동 `meeting` 일시정지, 종료 시 자동 재개, 회의 5분 전 짧은 휴식 알림
3. 집중 블록 제안 — 다음 일정까지 남은 시간 → 뽀모도로 회차 제안
4. 앱 사용 기록 — 앱 이름별 사용 시간 (창 제목 미수집)

## 이 페이즈에서 하지 않는 것

- 일정 생성·수정 → 범위 밖
- AI 요약 → P2 (P1 데이터를 P2 툴이 사용)

## 작업 체크리스트

### A. Swift 수집기

- [ ] **T1.A.1** — 캘린더 수집
  - 파일: `helpers/Sources/MenuBarApp/CalendarCollector.swift`, HelperCore `CalendarModels.swift`
  - 내용: 오늘 00:00 ~ 내일 24:00 일정. 필드: `id, title, start, end, all_day, has_attendees, availability, calendar_id`. `EKEventStoreChanged` + 5분 주기로 엔진 `context.calendar.update` RPC 호출. 설정에서 제외할 캘린더 선택.
  - 검증: 모델 인코딩 테스트, 수동으로 일정 추가 후 1분 내 팝오버 반영
- [ ] **T1.A.2** — 앱 사용 수집
  - 파일: `helpers/Sources/MenuBarApp/AppUsageCollector.swift`, 집계 순수 함수는 HelperCore
  - 내용: `NSWorkspace.didActivateApplicationNotification`으로 전면 앱 전환 기록, 화면 잠금/잠자기 시 구간 종료. 1분마다 `{bundle_id, name, seconds}` 증분을 `context.app_usage.add` RPC로 전송. 설정 `app_usage_enabled`(기본 true, 수집 범위를 설정 화면에 명시)
  - 검증: `swift test` 집계 테스트

### B. 엔진 판단 로직 (순수 함수)

- [ ] **T1.B.1** — `internal/workcontext`
  - 파일: `calendar.go`, `appusage.go`, 테스트
  - 내용: 일정 스냅샷 보관(메모리 + `paths` 아래 캐시), `CurrentMeeting(events, now, rule)`(종일·free·참석자 없는 개인 일정 제외 규칙은 설정 `meeting_rule`), `NextEvent`, `FocusWindow(events, now, cfg) (minutes, pomodoros)`. 갱신이 15분 넘게 없으면 "알 수 없음"으로 판단 보류. 앱 사용은 일별 JSON에 집계 저장, 30일 보관.
  - 검증: `go test ./internal/workcontext` (회의 시작/끝 경계, 겹치는 일정, 자정 넘김)
- [ ] **T1.B.2** — 회의 인식 휴식
  - 파일: `internal/engine`(Tick 루프), `internal/state/state.go`, `internal/timer`
  - 내용: 회의 중이고 수동 일시정지가 아니면 기존 `meeting` 일시정지 경로 사용, `PauseSource=calendar`와 종료 시각 저장. 종료 후 자동 재개. 사용자가 수동 재개하면 그 회의 동안 다시 자동 일시정지하지 않음. 회의 5분 전 & 연속 작업 ≥ 25분이면 짧은 휴식 알림 1회. 설정 `calendar_aware_breaks`(기본 false, 1주 사용 후 결정).
  - 주의: `state.Save()`/Load 양쪽 반영, `EnterBreak()` 경로 유지
  - 검증: `go test ./internal/timer ./internal/engine -run Meeting`
- [ ] **T1.B.3** — `status.get`에 `context.calendar{current,next,focus_minutes,focus_pomodoros,stale}`, `context.app_usage_today`(Top 5) 추가

### C. 팝오버 UI

- [ ] **T1.C.1** — "지금" 카드: 진행 중 회의 / 다음 일정과 남은 시간 / "다음 회의까지 52분 · 뽀모도로 2회" + [시작]. 권한 없음·데이터 오래됨 문구 포함
- [ ] **T1.C.2** — 오늘의 앱 사용 Top 5 (팝오버 접기 영역, Dashboard 통계 탭)
- [ ] **T1.C.3** — 설정: 캘린더 선택, 회의 인식 휴식, 앱 사용 기록 on/off·데이터 삭제
  - i18n: `catalog.json` 추가 후 `python3 scripts/generate-localizations.py`

### D. 진단

- [ ] **T1.D.1** — `doctor`(및 팝오버 진단 섹션)에 캘린더 권한·마지막 갱신 시각·앱 사용 수집 상태

---

## ✅ Phase 1 Checkpoint

**자동:** `make test`, `go test -race ./internal/workcontext ./internal/engine`

**수동:**
- [ ] 10분 뒤 테스트 회의 → 팝오버에 "10분 후"와 집중 가능 시간
- [ ] 회의 동안 휴식 화면 미표시, 종료 후 자동 재개
- [ ] 회의 중 수동 재개 → 그 회의 동안 다시 일시정지 안 함
- [ ] 캘린더 권한 거부 상태에서도 기존 휴식 기능 정상
- [ ] 앱 사용 Top 5가 실제 사용과 대략 일치

**완료 처리:** 1주 실사용 후 회의 인식 휴식 기본값과 회의 판정 규칙을 사용자와 확정하고 P2 진행.
