# Phase 0: 기반 — 작업지시서

_작성일: 2026-10-20 (rev.2) · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 32–48h_

## 페이즈 목표

이후 모든 기능이 올라갈 구조를 앱스토어 샌드박스에서도 통하는 형태로 바꾼다. 끝나면 Developer ID로 서명된 메뉴바 `.app`이 Go 엔진 사이드카를 띄우고, 팝오버에서 기존 휴식·뽀모도로·타이머가 그대로 동작하며, 샌드박스에서 막히는 지점이 스파이크 문서로 정리된다. 기존 launchd 설치 경로는 P4까지 그대로 둔다.

## 전제 조건

- [x] `make test`, `make build`가 현재 main에서 통과
- [x] Developer ID Application 인증서가 키체인에 있음 (사용자 보유 확인됨)

## 이 페이즈에서 하지 않는 것

- 캘린더 표시·휴식 연동 → P1
- AI/MCP → P2
- 이름 변경, launchd 구조 제거, 배포 → P4
- 앱스토어 제출 → P5

## 작업 체크리스트

### A. 엔진 사이드카

- [x] **T0.A.1** — 경로 일원화
  - 파일: `internal/paths/paths.go`, 테스트
  - 내용: 설정·상태·time-tools·AI 히스토리·인사이트·로그 경로를 한 곳에서 계산. `BREAK_REMINDER_HOME` 환경변수가 있으면 그 아래로(앱스토어판은 앱이 컨테이너 경로를 넘김), 없으면 기존 경로 그대로. 기존 하드코딩(`state`, `timetools.DefaultDirectory`, `ai.HistoryPath`, `insights.Path`, `config`, `logging`)을 교체.
  - 검증: `go test ./...` 기존 테스트 전부 통과 + 환경변수 지정 시 경로 테스트
- [x] **T0.A.2** — 엔진 명령과 RPC
  - 파일: `cmd/break-reminder/engine.go`(`engine --stdio`), `internal/engine/{server.go,methods.go,events.go}`, 테스트
  - 내용: 기존 `daemon` 루프의 `timer.Tick`을 재사용해 주기 실행. 줄 단위 JSON-RPC 2.0. 메서드: `status.get`, `session.start/stop`, `timer.pause/resume/reset/snooze`, `pomodoro.set`, `time_tools.*`(기존 cmd 로직 함수 재사용), `idle.report`(앱이 유휴 초를 보고). 알림: `event` 스트림(`break_start`, `break_over`, `five_min_warning`, `session_start/end`, `speak`, `time_tool_done`).
  - 참조: linetta `engine/cmd/linetta-engine`(stdio JSON-RPC), 기존 `cmd/break-reminder/check.go`의 `executeActions` → 이벤트 변환
  - 검증: 가짜 시계·가짜 유휴값으로 RPC 왕복 + 휴식 진입 이벤트 발생 테스트, stdout에는 JSON만
- [x] **T0.A.3** — `check`·워커와의 공존
  - 내용: 엔진 실행 중에는 launchd `check`가 상태를 건드리지 않도록 엔진이 `paths` 아래 lock 파일을 잡고 `check`는 lock이 있으면 즉시 종료. 기존 time-tools 워커도 같은 규칙.
  - 검증: 두 프로세스 동시 실행 테스트

### B. Swift 앱 (MenuBarApp → .app)

- [x] **T0.B.1** — 엔진 클라이언트
  - 파일: `helpers/Sources/HelperCore/EngineClient.swift`, `EngineModels.swift`, 테스트
  - 내용: 번들 내 `Contents/MacOS/engine`을 `Process`로 실행, stdin/stdout JSON-RPC, 비정상 종료 시 백오프 재시작, 앱 종료 시 정리
  - 참조: `CLICommandClient.swift`의 타임아웃·오류 처리
- [x] **T0.B.2** — OS 실행기(이벤트 → 동작)
  - 파일: `helpers/Sources/MenuBarApp/ActionExecutor.swift`
  - 내용: 알림은 `UNUserNotificationCenter`, 음성은 `AVSpeechSynthesizer`(설정의 voice 이름 매핑), 휴식 화면은 번들 내 `break-screen` 실행(P4에서 앱 프로세스로 통합 검토), 유휴 시간은 `CGEventSource.secondsSinceLastEventType`으로 측정해 `idle.report`
- [x] **T0.B.3** — 팝오버 셸
  - 파일: `StatusItemController.swift`, `PopoverRootView.swift` (기존 `main.swift` 분리)
  - 내용: 왼쪽 클릭 = SwiftUI 팝오버(상태 카드, 뽀모도로·타이머 카드 — Dashboard의 `PomodoroControlsView`·`CountdownView`를 HelperCore로 옮겨 공유), 오른쪽 클릭 = 기존 `NSMenu`. 데이터는 엔진 이벤트 기반 갱신.
- [x] **T0.B.4** — 번들·서명
  - 파일: `scripts/build-app.sh`, `helpers/Resources/App-Info.plist`, `helpers/Resources/{app,engine}.entitlements`, `Makefile`(`make app`)
  - 내용: `LSUIElement`, 캘린더 사용 설명, Hardened Runtime. 엔진 → 헬퍼 → 앱 순서로 Developer ID 서명. 로그인 시 실행은 `SMAppService.mainApp`. 개발 중에는 기존 바이너리 이름 유지.
  - 참조: linetta `apps/desktop/src-tauri/entitlements`, 서명 순서(사이드카 inherit → 앱)

### C. 샌드박스 스파이크 (결정 문서가 산출물)

- [x] **T0.C.1** — Developer ID + `app-sandbox` 엔타이틀먼트로 로컬 샌드박스 빌드를 만들어 확인 (linetta 검증 방식):
  - EventKit 권한 요청·유지(재빌드·재로그인 후)
  - `CGEventSource` 유휴 시간
  - 전체화면 오버레이 멀티모니터
  - 사이드카 실행, 컨테이너 경로 쓰기
  - `SMAppService` 로그인 항목
  - 127.0.0.1 HTTP 서버(P2 외부 MCP용)
  - 다른 프로세스 CPU/메모리 조회(P3 진단용)
  - 확인 방법: `log stream --predicate 'sender == "sandboxd"'`에 위반 없음
  - 산출물: `docs/plans/renewal-spike-sandbox.md` — 항목별 PASS/FAIL, FAIL 항목의 대안과 해당 페이즈 수정 사항

### D. 비밀값 정리

- [x] **T0.D.1** — Keychain 저장소: `internal/secrets`(직접판은 `security` 대신 Security.framework cgo 또는 Swift 쪽 저장 후 RPC 전달 — 스파이크 결과로 결정), `tts_api_key`를 이전
- [x] **T0.D.2** — SEC-002: 설정 파일 0600, `config show/get`에서 비밀값 마스킹
- [x] **T0.D.3** — SEC-001: `ai configure`가 프롬프트에 키를 넣지 않음

---

## ✅ Phase 0 Checkpoint

**자동:** `make test`, `go test -race ./internal/engine ./internal/paths`, `cd helpers && swift test`

**수동:**
- [ ] `make app`으로 만든 `.app` 실행 → 팝오버에 작업/휴식 진행률·타이머 표시, 일시정지/재개 동작
- [ ] 50분(테스트 설정 1분) 후 휴식 화면·알림·음성이 앱 경로로 나옴
- [ ] 엔진 강제 종료 → 앱이 재시작하고 상태 유지
- [ ] 엔진 실행 중 launchd `check`가 상태를 바꾸지 않음
- [ ] `codesign --verify --deep --strict`, `spctl -a -vv` 통과
- [ ] 스파이크 문서의 모든 항목이 PASS/FAIL로 채워짐

**완료 처리:** 스파이크 FAIL 항목과 그에 따른 로드맵 변경안을 사용자에게 보고하고 승인 후 P1 진행.
