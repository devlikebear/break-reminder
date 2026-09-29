# Phase 4 — 특정 날짜·시각에 울리는 1회 알람

로드맵: [time-tools-roadmap.md](time-tools-roadmap.md) · 계약: [time-tools-design.md](time-tools-design.md)

예상 16–24시간. Phase 2 worker/event를 확장한다. 최대 10개 예약, 이름·날짜·시각·현재 시간대, 수정/취소/완료 확인이 범위다. 반복·요일·snooze·캘린더 연동은 제외한다.

## 시작 게이트

첫 배포 1주 사용 확인 후 상대 타이머로 해결되지 않는 시각 예약 수요와 로드맵의 Phase 4 go/no-go를 확인한다. 필수 기술 선행은 Phase 2 worker/event이며 Phase 3 스톱워치는 필수가 아니다.

## 입력과 CLI

GUI는 날짜 선택 + 시/분 입력 + 시간대 표시. 과거이면 인라인 오류로 시작을 막는다. CLI는 `time-tools alarm add --at <RFC3339-with-offset> --label <name>`과 list/update/cancel을 제공한다. 초는 0으로 제한해 GUI와 계약을 통일한다. 화면은 같은 절대 시각을 현재 시간대로 보여준다.

CLI의 timezone metadata는 별도 `--time-zone <IANA-name>`로 받는다. 주어지면 RFC3339 offset과 해당 날짜의 zone offset이 일치하는지 검증한다. 생략하면 입력 offset을 원래 시간대 정보로 저장한다. offset 있는 timestamp이므로 CLI 자체는 DST occurrence가 모호하지 않다. GUI의 local 입력 변환은 nonexistent/ambiguous 시각을 별도로 검증한다.

## 작업 목록

- [ ] **P4.1 / 2–3시간 — 시간 계약 RED.** `internal/timetools/alarm_test.go`: 미래/과거/현재 동일 시각, 10개/11개, 서로 다른 offset이 같은 instant, DST gap/fold fixture. 테스트는 개발자 컴퓨터의 현재 TZ에 의존하지 않는다.
- [ ] **P4.2 / 2–3시간 — 모델/CLI.** `alarm.go`와 `time_tools.go`: scheduled/fired/canceled, update/cancel 경계에서 due 먼저 처리, 잘못된 시간은 거부. worker guard와 revision/ID 검사 재사용. scheduled 10개 제한이며 fired/canceled 정리는 이벤트 보존 정책에 따른다.
- [ ] **P4.3 / 3–5시간 — worker batch.** `scheduler.go/notification.go` 테스트 먼저 확장. 같은 tick의 여러 알람을 batch ID로 묶고 event별 완료를 기록한다. 전송 1회 결과는 batch 구성 event에 반영한다. 일반 타이머와 같은 시각이면 각 완료 기록을 남기고 요약 알림에 모두 표시한다.
- [ ] **P4.4 / 2–3시간 — migration/실패.** 이전 store fixture에 alarms 추가, 타이머·스톱워치·history·미확인 완료 보존 테스트. 프로세스 재시작, 5분 이상 늦은 예약, claimed 중 crash는 공통 규칙 적용. 시계 앞으로/뒤로 이동에도 fired event가 다시 scheduled로 돌아가지 않아야 한다.
- [ ] **P4.5 / 3–5시간 — UI.** 신규 `AlarmListView.swift`, `AlarmEditorView.swift`와 HelperCore 검증 모델. 다음 알람 메뉴 항목·대시보드 목록, 수정·취소, past 오류, 10개 제한, worker unavailable 안내. DST fold면 두 offset 중 선택, gap이면 다른 시각 입력 요청. 선택한 zone을 확정하기 전 암묵적 시간 보정 금지.
- [ ] **P4.6 / 4–5시간 — QA/문서.** 일반 타이머/스톱워치/휴식 동시 실행과 A01–A06 테스트. 원래 시간대·현재 시간대 표시, 늦은 완료, 서비스 종료 설명을 README에 추가.

## Checkpoint

- [ ] `go test ./internal/timetools ./internal/notify ./cmd/break-reminder`, `swift test --package-path helpers`, 최종 `make test`, `make build`.
- [ ] 실제 2분 뒤 알람을 예약하고 창을 닫은 뒤 worker 완료·시스템 알림 요청·다시 열린 UI의 완료 확인.
- [ ] 동일 시각 알람 2개: 완료 기록 2개, 요약 전송 1회. 완료를 확인한 뒤 재시작해도 재발송 없음.
- [ ] 생성 이후 표시 시간대를 바꿔도 동일 UTC instant에 실행됨. 실제 시스템 시계를 바꾸지 않고 주입 clock/TZ fixture로 자동 검증.
- [ ] sleep 중 지나간 알람이 5분 정책을 지킨다. Mac을 깨워서 울린다고 설명하지 않는다.
- [ ] 예약 10개 상태에서 11번째 거부, 기존 10개 보존. cancel/update 경합으로 다른 ID 알람이 수정되지 않는다.

전체 로드맵 완료 시 기능·자동 테스트·수동 QA·남은 제한을 함께 보고한다. 릴리스는 별도 승인 범위에 따라 기존 workflow로 수행한다.
