# 시간 도구 확장 — 수용 기준과 검증 기록 양식

작성일: 2026-09-28. 이 문서는 계획 단계의 수용 기준이다. 구현 후 실제 PASS/미검증 구분은 [검증 기록](time-tools-validation.md)을 따른다. 아래 체크박스를 전체 통과 표시로 해석하지 않는다.

## 테스트 환경

- Go unit/integration은 `t.TempDir()`와 주입 경로·시간·알림 adapter를 사용한다. 실제 HOME, launchctl, terminal-notifier를 자동 테스트에서 호출하지 않는다.
- 새 `time-tools` leaf 명령에 `--data-dir` 개발/테스트 옵션을 설계해 store/lock/runtime을 임시 디렉터리로 돌린다. 기본 사용자 경로와 혼용 금지.
- 새 Swift 시간 도구 adapter는 테스트 전용 data directory를 주입 가능하게 한다. 생산 기본값은 고정 사용자 경로. 전체 GUI의 기존 업무 config/state까지 검증할 때는 별도 macOS 테스트 계정 또는 명시적 path injection을 사용한다. 새 data-dir만 바꿨다고 기존 GUI 쓰기도 격리됐다고 가정하지 않는다.
- 기존 config 검증을 위한 임시 HOME은 Go 테스트에서만 사용한다. Swift `homeDirectoryForCurrentUser`가 shell HOME과 일치한다고 가정하지 않는다.
- 실제 launchd install/upgrade/uninstall QA는 테스트 계정에서 실행한다. 본 사용자 서비스/설정을 덮어써서 검증하지 않는다.

## 제품 수용 기준

| ID | Given / When | Then | 검증 방식 |
|---|---|---|---|
| P01 | stopped에서 포모도로 시작 | 모드 저장 후 수동 세션 시작, 두 UI 동일 | mock CLI + UI |
| P02 | 활동 pomodoro 4회 완료 | 짧은 휴식 3회·긴 휴식 1회, 일일 횟수 정확 | Go fake clock |
| P03 | on 성공, start 실패 | 부분 성공 안내, start만 재시도 | controller test |
| P04 | pause/break 중 모드 전환 | GUI 전환 disabled, 현재 상태 유지 | presentation + UI |
| P05 | label/설정 입력 중 q/r/b | 텍스트 입력만 되고 quit/reset/break 없음 | policy test + UI |
| T01 | 빈 상태에서 10초 시작 | deadline 저장, 두 UI 동일 초 표시 | model/CLI/UI |
| T02 | 10초 중 4초 후 pause, 30초 후 resume | 남은 약 6초로 재개 | fake clock |
| T03 | paused/running에서 두 번째 start | conflict, 명시 replace 전까지 보존 | CLI |
| T04 | deadline 전 cancel | 완료 이벤트·알림 없음 | model/worker |
| T05 | 정확히 deadline에 pause/cancel | 완료 우선, 이벤트 1개 | boundary test |
| T06 | A 취소 후 B 시작, 다른 창이 옛 A의 ID/revision으로 pause | conflict/not_found, 새 B는 running 유지. 같은 revision 동시 변경도 하나만 성공 | multiprocess + UI |
| T07 | 창·메뉴바 닫고 deadline 도달 | worker가 completed 저장·알림 요청 | 실제 runtime |
| T08 | worker crash/restart, deadline ≤300초 지남 | 완료 복원, 미시도 event만 자동 전송 | integration |
| T09 | sleep/서비스 중지 후 >300초 지남 | silent 완료, UI에 지난 완료 표시 | fake + runtime |
| T10 | 알림 명령 실패 또는 claim 후 crash | failed/unknown, 자동 중복 시도 없음 | injected notifier |
| T11 | store 손상/unsupported schema | 원본 보존, 오류 표시, 시작 차단 | store + UI |
| T12 | 업무 stop/주말/휴식 overlay 중 실행 | deadline 정책대로 완료, 업무 상태 독립 | integration + runtime |
| T13 | notifier 없음 또는 실행 실패 | 시작 전 경고, 완료 event·메뉴바·카드 유지, doctor 복구 안내; 소리는 보장하지 않음 | injected notifier + UI |
| T14 | primary 휴식 overlay 중 배너가 가려짐 | 읽기 전용 완료 안내 표시, 가이드/휴식 유지, 휴식 후에도 미확인 완료 보존 | runtime + screenshot |
| T15 | tick 없이 벽시계만 10분 이동 후 reconcile 1회 | 저장한 deadline 기준 즉시 완료·5분 정책 적용 | fake wall clock |
| T16 | 기존 업무 Reset/Force Break 실행 | 별도 time-tools JSON의 ID·상태·revision 유실 없음 | integration + UI |
| U01 | 구버전 updater 첫 upgrade 또는 직접 brew upgrade | worker 미설치 정확히 표시, 1회 설정 전 start 차단, 명시 설치 후 실행 | old-plist fixture + runtime |
| U02 | 새 updater가 새 버전 설치 | 새 바이너리로 migration, idempotent, updater 자신은 reload하지 않음 | injected process + service tests |
| U03 | 기존 runtime stopped/uninstalled 상태에서 자동 upgrade | 의도치 않은 agent 설치/load 없음, 예약 데이터 보존 | service tests |
| U04 | KeepAlive worker crash/명시 stop/logout-login | crash 재시도, stop 후 유지 종료, login 후 startup reconcile | isolated runtime |
| U05 | migration 일부 실패 후 재실행 | 성공 marker 조기 저장 없음, 중복 job 없이 복구, 데이터 보존 | failure injection |
| S01 | 10초 실행·5초 pause·10초 실행 | 약 20초 측정, pause 제외 | fake + UI |
| S02 | 두 UI가 finish 동시 요청 | 세션당 history 1개 | multiprocess |
| S03 | worker 없음/창 종료 후 복귀 | 스톱워치 계속 경과·저장 가능 | CLI + runtime |
| S04 | 기록 삭제/reset | 지정 기록만 삭제, 업무 통계 불변 | integration |
| S05 | 기록 501번째 저장 | 오래된 1개 정리, 최신 500개 유지 | store |
| A01 | 과거 예약/11번째 active 예약 | validation 오류, 기존 예약 보존 | model/CLI |
| A02 | 현재보다 2분 뒤 예약 후 창 닫기 | 예정 instant 도달 후 완료·전송 | runtime |
| A03 | 같은 시각 예약 2개 | 이벤트 2개, 요약 전송 1회 | worker |
| A04 | 표시 시간대 변경 | 예약 UTC instant 불변 | fixture + UI |
| A05 | DST 없는 시각/중복 시각 입력 | 거부 또는 offset 선택, 암묵적 보정 없음 | fixture + UI |
| A06 | 이미 fired 후 시계 역행/restart | 같은 알람 재발화 없음 | fake clock |

## 공통 기술 검증

- [ ] store 쓰기 권한/디스크 오류, lock contention, temp 파일 남음, 동시 별도 프로세스 갱신.
- [ ] ack가 전송보다 빠를 때 silent, 알림 timeout 중 다른 완료가 계속 저장됨.
- [ ] 새 leaf 명령은 업무 config가 손상돼도 조회/취소/worker 실행 가능.
- [ ] worker 없는 start/resume은 실패하고 snapshot 불변. 스톱워치는 예외로 허용.
- [ ] UI 1초 refresh에서 CLI spawn/데이터 쓰기 없음. worker 변경 없는 tick에서 snapshot 쓰기 없음.
- [ ] schema migration은 기존 도구/이벤트/최근 설정을 보존하고 미래 schema를 자동 덮어쓰지 않음.
- [ ] upgrade RestartRuntime과 service stop/uninstall은 worker를 포함하고 예약 파일은 보존.
- [ ] terminal-notifier의 event group이 기존 휴식 group과 충돌하지 않음.

## 수동 UI·품질 검증

- [ ] 메뉴바 1x/2x, 밝은/어두운 모드에서 숫자·아이콘 식별 가능. 메뉴를 15초 열어도 갱신 지속.
- [ ] 긴 한글 이름, 80자 경계, emoji, control character 입력, 1초/24시간 경계.
- [ ] 360×600에서 필드·오류·완료 버튼이 잘리지 않음. scroll과 키보드 포커스 유지.
- [ ] VoiceOver는 이름·상태를 읽으며 초마다 announcement를 남발하지 않음.
- [ ] 기본 휴식, 긴 휴식, guided stretch, 기존 reset/force break/pause/resume 회귀.
- [ ] 알림 권한/집중 모드로 배너가 안 보이는 경우에도 완료 카드와 확인 버튼이 유지됨.
- [ ] primary 전체화면의 ready/running/completed 가이드 각각에서 타이머 완료: 안내 줄이 가려지지 않고 기존 버튼/레이아웃/키보드 조작을 방해하지 않음. secondary 화면 동작 유지.
- [ ] 화면 잠금 중 가시성을 보장하지 않으며 unlock 후 지속 완료 표시를 확인. UI를 모두 닫고 notifier까지 없을 때 즉시 소리/배너가 없다는 제한을 문서와 대조.
- [ ] sleep 전 남은 시간만큼 복귀 후 다시 기다리지 않음: 실제 wake 시각·첫 reconcile·completed 저장 시각을 각각 측정한다.
- [ ] UI 표시 갱신 목표는 정상 활성 환경에서 1초, worker 완료 목표는 deadline 후 2초 이내. 부하·sleep 환경은 별도 복구 기준 적용.

## 구현 시 검증 명령

```bash
go test ./internal/timer ./internal/state ./cmd/break-reminder
go test ./internal/timetools ./internal/notify ./internal/launchd
go test -race ./internal/timetools
swift test --package-path helpers
make test
make build
```

`internal/timetools` 등 신규 경로 명령은 해당 단계에서 생성한 이후에 실행한다. 같은 변경에 대해 전체 테스트를 반복 실행하는 대신 관련 RED/GREEN 후 최종 전체 검증을 한 번 수행한다. 추가 실패/수정이 있으면 해당 검증을 다시 한다.

## 검증 기록 양식

| 항목 | 기록 |
|---|---|
| 구현 commit / OS / CPU architecture | 구현 시 작성 |
| 단계 / 테스트 ID | 구현 시 작성 |
| 명령·환경·clock 기준 | 실제 명령 및 격리 방법 |
| 기대 / 실제 | 측정된 값, 상태, 전송 요청 횟수 |
| 증거 | 로그/스크린샷/테스트 출력 파일 경로 |
| 판정 | PASS / FAIL / NOT RUN |
| 제한과 후속 | 미검증을 포함해 기록 |

단계별 결과는 `docs/plans/time-tools-validation.md`에 구현 중 새로 기록한다. 이 파일을 계획 단계에서 PASS로 미리 채우지 않는다.
