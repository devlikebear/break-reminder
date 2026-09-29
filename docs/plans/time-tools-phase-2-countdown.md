# Phase 2 — 독립 일반 타이머와 백그라운드 완료

로드맵: [time-tools-roadmap.md](time-tools-roadmap.md) · 계약: [time-tools-design.md](time-tools-design.md)

예상 32–50시간. 설치/업그레이드 전환 5–8시간과 알림 fallback/overlay 연동 증가분을 포함한다. Phase 1과 함께 첫 배포 범위. 이 단계가 끝나면 이름 있는 타이머 하나를 GUI/CLI에서 조작하고 창을 닫아도 완료를 확인할 수 있다. 별도 worker는 이 사용자 기능을 완결하는 데 필요한 부분이다.

## CLI 제안 계약

아래 명령은 **구현 예정**이며 현재 실행 가능한 명령이 아니다.

```text
break-reminder time-tools status --json
break-reminder time-tools timer start --duration 15m --label "코드 리뷰" --if-revision 0 --json
break-reminder time-tools timer pause --id <id> --if-revision <rev> --json
break-reminder time-tools timer resume --id <id> --if-revision <rev> --json
break-reminder time-tools timer cancel --id <id> --if-revision <rev> --json
break-reminder time-tools timer restart --id <id> --if-revision <rev> --json
break-reminder time-tools timer start --duration 5m --replace-id <id> --if-revision <rev> --json
break-reminder time-tools acknowledge --event-id <event-id> --json
break-reminder time-tools notify-again --event-id <event-id> --json
break-reminder time-tools run
```

GUI는 revision과 ID를 항상 전달한다. 사람용 CLI에서 revision 생략은 허용하되 활성 타이머 충돌/명시적 replace 규칙은 동일하다. 모든 변경은 success JSON `{schema_version, snapshot, runtime}` 또는 nonzero+JSON error `{code,message}`를 반환한다. error code는 `invalid_input`, `conflict`, `not_found`, `worker_unavailable`, `store_corrupt`, `unsupported_schema`, `io_error`로 고정한다. stdout에 로그/진행문구를 섞지 않는다.

## 작업 목록

### P2.1 순수 모델과 상태 전이 — 3–5시간

- [ ] 신규 `internal/timetools/countdown_test.go`에 start/pause/resume/cancel/replace/만료/정확한 경계/중복 명령 RED.
- [ ] `model.go`, `countdown.go`: `Apply(snapshot, command, now)`와 `Reconcile(snapshot, now)`의 결정적 전이 구현. UUID 생성도 주입 가능하게 한다. 함수는 IO/알림을 하지 않는다.
- [ ] duration은 Go `time.ParseDuration` 후 1초~24시간, 1초 단위 값만 허용. integer overflow와 부호·잘못된 단위 거부. label은 공통 계약 적용.
- [ ] UTC deadline과 paused remaining은 ms 정밀도로 저장하고 UI는 ceil 초로 표시한다. 마지막 0초는 completed 상태로 표현한다.
- [ ] 검증: `go test ./internal/timetools -run 'TestCountdown|TestReconcile'`.

### P2.2 영속 저장·경합 — 3–5시간

- [ ] `store_test.go` RED: 파일 없음, round-trip, 0600, corrupt/지원 밖 버전 보존, revision 충돌, 저장 실패 시 기존 파일 보존.
- [ ] `store.go`에 flock과 atomic rename 구현. read-modify-write 전체를 잠근다. crash 후 남은 temp 파일은 정식 snapshot으로 읽지 않는다.
- [ ] 두 별도 프로세스가 동시에 pause/cancel/start한 경우도 테스트해 goroutine race만 검사하는 데 그치지 않는다.
- [ ] 이벤트 생성과 완료 상태를 한 snapshot으로 저장. ack/recent 설정 변경도 동일 store 사용. 시각·ID·IO를 주입하여 테스트가 실제 HOME을 쓰지 않게 한다.
- [ ] schema_version v1 기본 필드·상한·unknown field 보존 정책 확정. store unit test로 후속 field round-trip 유실 방지.

### P2.3 CLI와 JSON 계약 — 2–4시간

- [ ] 신규 `cmd/break-reminder/time_tools_test.go` RED와 `time_tools.go` 구현, `main.go` 명령 등록.
- [ ] `time-tools --data-dir <path>`를 parent persistent flag로 제공하고 모든 leaf의 store/lock/runtime 경로에 동일 적용. 테스트는 이 경로 밖 쓰기가 없는지 검사한다. Swift adapter에도 동일한 경로 주입 지점을 둔다.
- [ ] 모든 leaf 명령에 invalid 업무 config 우회 설정. status를 사용하기 위해 업무 config를 고치게 만들지 않는다.
- [ ] 표준 출력 JSON 구조, nonzero 오류, target ID/revision 검증, stale UI command 거부를 고정 fixture로 테스트한다.
- [ ] 타이머 duration 이름과 JSON timestamp 단위를 문서·Swift 양쪽에서 통일한다.

### P2.4 worker와 알림 — 4–6시간

- [ ] `scheduler_test.go` RED: 가짜 시간으로 due 1회, restart recovery, 5분 경계(300초 포함/초과), ack-before-send, claimed crash, 알림 실패.
- [ ] `scheduler.go`: 약 1초마다 새 `time.Now().UnixMilli()`로 deadline 비교, startup 즉시 reconcile. 장시간 Timer/Sleep 한 번으로 완료 처리 금지. tick이 오지 않는 동안 벽시계만 10분 전진한 뒤 한 번 reconcile하는 sleep 재현 RED 추가. single-instance lock, SIGTERM/context 종료, 변경 없는 snapshot 쓰기 금지, heartbeat 최대 5초 간격.
- [ ] 알림 전송 큐는 reconcile과 분리. 5초 timeout, shutdown 시 미처리 pending은 보존. 실행 중 claim은 재시작 후 unknown 처리. retry는 명시적 사용자 동작만 허용.
- [ ] `internal/notify`에 event group을 받는 추가 API와 테스트. 기존 고정 group `Send`는 하위 호환 유지. notifier 실행 파일 없음/실패도 완료 기록은 유지한다.
- [ ] health unavailable일 때 start/resume 거부, 상태 조회·취소·확인 허용. health는 실제 알림 표시 보장이 아니라 worker 생존 정보임을 구분.

### P2.5 launchd·진단·업데이트 — 5–8시간

- [ ] `internal/launchd/launchd_test.go`와 `cmd/break-reminder/service_test.go` RED: 새 plist program args, install/start/stop/uninstall/status/RestartRuntime 포함, menuBar helper가 없어도 worker 설치.
- [ ] `launchd.go`에 TimeToolsLabel/plist/status 추가. `time-tools run`, RunAtLoad/KeepAlive, Aqua session 제한, ThrottleInterval=10을 검증하며 기존 timer StartInterval=60은 유지. Stop은 unload, crash는 재실행, logout/login은 복구 테스트.
- [ ] `service.go`, `cmd/break-reminder/update.go`, `internal/doctor`, `internal/autoupdate`의 관련 경로 연결. 새 updater는 stable bin의 새 바이너리에 `service migrate-runtime`을 호출한다. updater job 자기 reload 금지, 기존 stopped/uninstalled 상태 보존. `update_test.go`에서 migration 호출 순서·실패·재실행 RED 작성.
- [ ] 구버전 updater 최초 전환과 직접 brew upgrade는 자동 worker 설치를 보장하지 않음: `Formula/break-reminder.rb` 안내와 새 UI의 1회 `시간 도구 설정` 흐름 구현. 구버전과 같은 old-plist fixture에서 신규 worker 없음→진단→명시 설치→health 정상까지 검증.
- [ ] install 중간 실패 시 어느 job이 남았는지 반환하고 재실행으로 복구 가능하게 한다. uninstall은 새 JSON을 삭제하지 않는다.
- [ ] service stop/업데이트 재시작 시 active deadline 유지와 복구를 테스트한다. 실제 사용자 LaunchAgent에 대한 QA는 자동 테스트와 분리한다.

### P2.6 두 UI와 휴식 화면 연결 — 6–9시간

- [ ] `HelperCore/TimeToolsSnapshot.swift` Codable DTO, `TimeToolsPresentation.swift` 표시 함수 및 fixture 테스트 RED. 없는 파일/깨진 파일은 빈 정상 상태와 구분.
- [ ] `TimeToolsViewModel.swift` + `CountdownView.swift`: 입력/프리셋/최근 5개/상태/완료/에러/health, shared CLI client 사용.
- [ ] MenuBar `main.swift`와 `MenuBarPresentation.swift`: 타이머 하위 메뉴, 제목 우선순위, 완료 확인/재시작. 메뉴 tracking 동안 갱신되는 기존 timer 재사용.
- [ ] 1초 UI refresh는 snapshot 읽기와 projection만 한다. 매초 CLI 프로세스 실행·알림 발송·데이터 쓰기 금지.
- [ ] 두 UI의 pause/cancel 교차 조작에서 stale revision을 재읽고 사용자에게 표시. 특히 A 취소→B 새 시작→다른 창의 A pause가 B에 적용되지 않는 RED 작성. ID가 다르면 revision 생략 시에도 거부. 빠른 중복 클릭 차단.
- [ ] terminal-notifier 없음/실패 시 시작 전 경고와 지속 완료 카드/메뉴 표시 제공. 별도 소리 fallback은 없음을 설명. `doctor` 진단과 복구 안내 테스트.
- [ ] `HelperCore/TimeToolsPresentation.swift`에 overlay 완료 안내 projection 테스트 후 `BreakScreenApp/main.swift` primary 화면에 읽기 전용 완료 줄 연결. 기존 tick 재사용, state write/ack 금지. 휴식·스트레칭 흐름 중 완료해도 해당 흐름을 유지.
- [ ] 미설치/미실행 worker 복구 버튼은 명시적으로 `service install/start`를 실행하는 동작으로 제공하고 실패를 표시한다. 화면을 여는 것만으로 시스템 서비스를 설치하지 않는다.

### P2.7 통합 검증과 문서 — 3–5시간

- [ ] [수용 기준](time-tools-acceptance.md)의 T01–T16, U01–U05 및 공통 UI/서비스 시나리오 실행.
- [ ] README에 worker/서비스·독립 알림·sleep/5분 정책, CLI 예시, 날짜/시간 기준을 추가.
- [ ] CHANGELOG Unreleased에 첫 배포 범위를 작성. 버전 범프/릴리스는 이 단계 완료와 구분.

## Checkpoint

- [ ] `go test ./internal/timetools ./internal/notify ./internal/launchd ./cmd/break-reminder`.
- [ ] `go test -race ./internal/timetools` 및 별도 프로세스 잠금 테스트.
- [ ] `swift test --package-path helpers`, `make test`, `make build` 최종 통과.
- [ ] 실제 10초 타이머: 정상 깨어 있는 로컬 환경에서 deadline 이후 2초 이내 completed, 알림 프로세스 호출 1회. 배너 표시 시간은 별도 관측값으로 기록.
- [ ] 창/메뉴바 종료, 휴식 overlay 중, worker 재시작, sleep 복귀를 실환경에서 검증. 못 한 항목은 NOT RUN.
- [ ] 일반 타이머 작동 전후 기존 업무 state·config·통계가 변경되지 않음(기존 check에 의한 변화는 별도 baseline으로 구분).
- [ ] timer가 없는 10분과 실행 중 10분의 worker CPU/wakeups/메모리 관측. 1초 loop 외에 예상하지 않은 busy loop/프로세스 폭증/지속 증가가 있으면 수정. 실측을 확보하기 전 에너지 개선을 주장하지 않는다.

통과하면 **첫 배포 구현 범위 완료**를 보고한다. 이후 1주 사용 검증과 로드맵의 go/no-go 판정 전에는 3·4단계를 자동으로 시작하지 않는다. 결과에 따라 [Phase 3](time-tools-phase-3-stopwatch.md) 또는 Phase 4의 범위를 확정한다.
