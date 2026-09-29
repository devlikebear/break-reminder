# Phase 3 — 스톱워치와 로컬 작업 기록

로드맵: [time-tools-roadmap.md](time-tools-roadmap.md) · 계약: [time-tools-design.md](time-tools-design.md)

예상 12–20시간. Phase 2의 store/CLI client/UI 구조를 사용한다. 하나의 스톱워치를 시작·멈추고 기록하는 기능이며 기존 업무 통계 합산·랩·CSV export는 제외한다.

## 시작 게이트

첫 배포 1주 사용 확인 후 로드맵의 Phase 3 go/no-go를 통과해야 한다. 업무 통계 합산 없이도 측정 기록을 다시 볼 사용 사례를 기록한다. Phase 2 의존성은 store/CLI/UI 재사용이며 worker 실행은 전제조건이 아니다.

## 사용 흐름

대시보드 Timer → 스톱워치 → 이름 입력 → 시작 → 일시정지/재개 → 완료 및 저장 → 최근 기록. 메뉴바에서는 경과 시간과 시작/일시정지/재개/저장을 제공하며 기록 관리는 대시보드에서 한다. 메뉴바 제목은 기존 우선순위를 유지한다.

제안 CLI: `time-tools stopwatch start/pause/resume/reset/finish`, `time-tools history list/delete`. 변경 명령은 Phase 2와 동일한 ID/revision/JSON 계약이다. 기록 ID 없이 `delete all` 같은 일괄 삭제는 제공하지 않는다.

## 작업 목록

- [ ] **P3.1 / 2–3시간 — 모델 RED/GREEN.** `internal/timetools/stopwatch_test.go`에 start→pause→resume→finish, pause 시간 제외, sleep 포함, reset 후 0, 중복 finish 기록 1개를 테스트한다. `stopwatch.go`에 순수 transition 구현. 경과 시간은 `accumulated + max(0, now-startedAt)`이고 시스템 시계 역행은 0 아래로 내려가지 않는다.
- [ ] **P3.2 / 2–3시간 — 저장·호환.** `model.go/store.go`에 stopwatch와 최대 500개 history 추가. v1 fixture 마이그레이션, unknown schema 거부, 타이머/미확인 이벤트 보존, 저장 실패 시 running 세션 유지 테스트. session ID를 기록 ID로 사용해 finish 재시도를 멱등 처리한다. 저장 당시 local date/time zone을 보존한다.
- [ ] **P3.3 / 2–3시간 — CLI.** `time_tools.go`에 명령 추가하고 table-driven 테스트. stopwatch는 알림이 필요 없으므로 worker 없이 start/resume/finish를 허용한다. 일반 타이머의 worker guard를 공통으로 잘못 적용하지 않는다. finish/save/reset은 모두 atomic store mutation.
- [ ] **P3.4 / 2–4시간 — Swift.** HelperCore snapshot/presentation 테스트 후 `StopwatchView.swift` 추가. idle/running/paused의 버튼 상태, 1시간 이상 표시, 최근 기록·단건 삭제, 기록 500개 제한 안내. reset/기록 삭제는 취소 가능한 확인 UI. 같은 store의 running 타이머는 영향을 받지 않는다.
- [ ] **P3.5 / 2–3시간 — 통계 경계.** UI 항목명을 `측정 기록`으로 표시하고 기존 Work total과 섞지 않는다. 기존 history 파일·daily totals를 쓰지 않는 테스트를 추가한다. 타이머·스톱워치 동시 실행 후 종료 순서를 바꿔 두 기록이 독립인지 검증한다.
- [ ] **P3.6 / 2–4시간 — QA/문서.** 아래 checkpoint와 수용 기준 S01–S05를 실행하고 README/CHANGELOG를 갱신한다.

## Checkpoint

- [ ] `go test ./internal/timetools ./cmd/break-reminder`, `swift test --package-path helpers`, 최종 `make test`, `make build`.
- [ ] 10초 실행→5초 pause→10초 실행→저장: 실행 경과가 약 20초이며 pause 5초 제외(수동 조작 오차 기록).
- [ ] 창 종료 후 다시 열어 계속 증가한 값을 확인하고 worker가 없어도 표시/저장이 가능하다.
- [ ] 재시작·sleep을 포함한 경과 정책과 화면 설명이 일치한다.
- [ ] finish를 두 UI에서 동시에 요청해도 기록 한 개, 기존 타이머 계속 실행.
- [ ] 기존 업무 일일 누계에 측정 기록이 별도로 더해지지 않는다.

결과 보고 후 알람 사용 사례에 대한 별도 go/no-go와 승인된 범위에 따라 [Phase 4](time-tools-phase-4-alarm.md)로 진행한다. 실패한 테스트/QA는 완료로 처리하지 않는다.
