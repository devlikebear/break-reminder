# 시간 도구 Phase 1/2 구현 검증

검증일: 2026-09-28~29 · 기준: `8e8cb79`에서 분기한 `codex/time-tools` · macOS arm64.

이번 배포 범위는 포모도로 GUI와 독립 카운트다운이다. Phase 3 스톱워치와 Phase 4 알람은 구현하지 않았으며 첫 배포 후 1주 사용 결과로 각각 go/no-go를 결정한다.

## 자동 검증

| 검증 | 결과 |
|---|---|
| `make test` (Go 전체 + Swift 전체) | PASS. Go 전체 패키지, Swift 136개 테스트. 이후 controller 테스트 2개 추가 후 해당 suite 5개 PASS |
| `go test -race ./internal/timetools` | PASS. 상태 전이, 다중 프로세스 잠금, ID/revision 충돌, claim 복구, 알림 실패, worker 중복 실행·종료 |
| `make build` | PASS. release Swift helper 3개, resource bundle, Go binary |
| `git diff --check` | PASS |
| store | 별도 JSON, 원자적 교체, callback 실패 시 원본 보존, 미래 schema 거부·추가 필드 보존, 4프로세스 × 10변경 revision=40 검증 |
| CLI | 손상된 업무 config 독립, worker 미실행 시 시작 차단, duration 경계, stale revision 거부 |
| updater/service | 새 바이너리 migration 호출, 정지 서비스 보존, 신규 worker plist/Aqua/KeepAlive, 메뉴 helper 없는 설치 검증 |
| Pomodoro | 명시적 0/음수 거부, state reset 오류 보고, on→start 순서, 부분 성공·중복 클릭, 입력 단축키 보호 |
| 시계 역행 | heartbeat가 미래에 남아 worker를 중지로 오인하는 실패 테스트를 먼저 확인하고 수정 후 PASS |

테스트는 임시 Go/Swift 캐시와 임시 데이터 경로를 사용했다. 설치 단위 테스트는 launchctl 호출을 주입하여 사용자 서비스를 수정하지 않는다. 신규 동작은 구현 전 실패 테스트를 확인한 뒤 구현했으며 controller 회귀 테스트를 추가했다.

## 실제 실행 검증

- 격리 데이터 경로와 QA용 dashboard bundle로 10초 타이머 시작·완료 카드를 확인했다. 이름에 `qrb`를 입력해도 앱 종료·업무 reset·강제 휴식이 실행되지 않았다.
- 360×600 화면에서 일반 타이머에 기존 큰 업무 헤더가 공간을 과도하게 차지하는 문제를 확인하고, 해당 화면의 업무 상태를 작은 헤더로 바꾼 뒤 화면을 재확인했다.
- 알림 실행 실패 시 완료 기록과 재전송 버튼이 유지됐다. 정상 실행 환경의 worker로 명시적 재전송 후 `sent` 상태를 확인했다. 이는 전송 요청 성공이며 실제 OS 배너 노출을 의미하지 않는다.
- 완료 확인 후 다시 시작하고 dashboard를 Cmd+Q로 닫았다. deadline `1790590290197`, notification attempted `1790590291286` (Unix ms): 창이 닫힌 상태에서 약 1.089초 후 전송 요청이 기록됐다.
- 별도 임시 launchd label로 RunAtLoad/KeepAlive/Aqua worker를 실행했다. PID 6844 강제 종료 후 PID 6859로 재시작됐고 3초 타이머 완료가 복원됐다. `bootout` 후 2초 뒤 작업이 제거되고 runtime ready=false임을 확인했다. 사용자 agent는 수정하지 않았다.

## 미검증 및 제한

- 메뉴바 앱의 실제 메뉴 검증은 컴퓨터 제어 서버 `timeoutReached`로 완료하지 못했다. 1x/2x·light/dark, VoiceOver, 메뉴 열린 상태 refresh는 NOT RUN.
- primary/secondary 실제 전체화면에서 안내 줄과 guided stretch의 겹침, 실제 sleep/wake·logout/login, 잠금 화면은 NOT RUN. 시간 이동/늦은 복구는 fake clock과 worker 테스트로 검증했다.
- 기존 사용자의 실제 Homebrew upgrade→service install 전환 전체 과정은 NOT RUN. migration 호출 순서·정지 유지·plist 구조는 자동 테스트로 검증했다.
- 실제 Pomodoro 모드 전환은 사용자 업무 state를 변경하지 않기 위해 GUI에서 실행하지 않았다. 명령·controller·기존 엔진 테스트로 검증했다.
- 소리/배너는 terminal-notifier, OS 권한 및 집중 모드에 의존한다. 지속 완료 표시가 fallback이며 잠자기 중 알람이나 자동 wake는 보장하지 않는다.

위 미검증 항목을 PASS로 간주하지 않는다. 릴리즈 후 사용 검토에서 특히 메뉴바 가독성, 전체화면 안내, 실제 wake 복구를 확인한다.
