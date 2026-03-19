# 구현 체크리스트

각 항목은 독립적으로 검증 가능한 단위. 순서대로 진행하면 점진적으로 동작하는 시스템이 만들어짐.

---

## Phase 1: 코어 (필수)

- [ ] **P1-01** Go 모듈 초기화 + Cobra root 커맨드
  - 파일: `go.mod`, `cmd/break-reminder/main.go`
  - 검증: `go build ./cmd/break-reminder`

- [ ] **P1-02** Config 로드 (YAML + 기본값 병합 + boolean 머지)
  - 파일: `internal/config/types.go`, `defaults.go`, `load.go`
  - 검증: `go test ./internal/config/`

- [ ] **P1-03** State 저장/로드 (key=value 플랫 파일)
  - 파일: `internal/state/state.go`
  - 검증: `go test ./internal/state/` — 라운드트립 + 기존 형식 파싱

- [ ] **P1-04** timer.Tick() 순수 함수
  - 파일: `internal/timer/timer.go`, `timer_test.go`
  - 검증: 테이블 드리븐 테스트 — work→break, break→work, idle 리셋, 일간 리셋, 갭 감지

- [ ] **P1-05** 유휴 감지 (idle)
  - 파일: `internal/idle/idle.go`, `idle_darwin.go`, `idle_stub.go`
  - 검증: darwin에서 `IdleSeconds()` > 0

- [ ] **P1-06** 알림 (notify)
  - 파일: `internal/notify/notify.go`, `notify_darwin.go`, `notify_stub.go`
  - 검증: `Send("test", "message", "")` → macOS 알림 표시

- [ ] **P1-07** TTS
  - 파일: `internal/tts/tts.go`, `tts_darwin.go`, `tts_stub.go`
  - 검증: `Speak("Yuna", "테스트")` → 음성 출력

- [ ] **P1-08** check 커맨드 + executeActions
  - 파일: `cmd/break-reminder/check.go`
  - 검증: `break-reminder check` → state 파일 갱신

- [ ] **P1-09** status 커맨드
  - 파일: `cmd/break-reminder/status.go`
  - 검증: `break-reminder status` → 상태 출력

- [ ] **P1-10** reset 커맨드
  - 파일: `cmd/break-reminder/reset.go`
  - 검증: `break-reminder reset` → state 파일 초기화

- [ ] **P1-11** daemon 커맨드
  - 파일: `cmd/break-reminder/daemon.go`
  - 검증: `break-reminder daemon` → 60초 간격 체크 + Ctrl+C 정상 종료

---

## Phase 2: 근무 스케줄 + 서비스

- [ ] **P2-01** schedule (근무시간/요일 체크)
  - 파일: `internal/schedule/schedule.go`
  - 검증: 주말/업무외 시간에 `IsWorkTime()` == false

- [ ] **P2-02** LaunchAgent 관리
  - 파일: `internal/launchd/launchd.go`
  - 검증: plist 파일 생성 + `launchctl list | grep break-reminder`

- [ ] **P2-03** service 커맨드 (install/uninstall/start/stop/status)
  - 파일: `cmd/break-reminder/service.go`
  - 검증: `break-reminder service install && break-reminder service status`

- [ ] **P2-04** doctor 커맨드
  - 파일: `internal/doctor/doctor.go`, `cmd/break-reminder/doctor.go`
  - 검증: `break-reminder doctor` → 모든 체크 결과 출력

---

## Phase 3: UI

- [ ] **P3-01** TUI 대시보드 (Bubbletea)
  - 파일: `internal/dashboard/dashboard.go`, `cmd/break-reminder/dashboard.go`
  - 검증: `break-reminder dashboard` → 실시간 프로그레스 바 + 키 바인딩 동작

- [ ] **P3-02** Swift SPM 패키지 구조 + HelperCore
  - 파일: `helpers/Package.swift`, `helpers/Sources/HelperCore/*.swift`
  - 검증: `swift test --package-path helpers/`

- [ ] **P3-03** BreakScreenApp (풀스크린 잠금)
  - 파일: `helpers/Sources/BreakScreenApp/main.swift`
  - 검증: `./bin/break-screen --duration 10` → 풀스크린 오버레이 + 카운트다운

- [ ] **P3-04** breakscreen 패키지 (Go ↔ Swift 연동)
  - 파일: `internal/breakscreen/breakscreen.go`, `overlay_darwin.go`, `overlay_stub.go`, `ask_darwin.go`
  - 검증: config `break_screen_mode: block` → 풀스크린 표시

- [ ] **P3-05** DashboardApp (네이티브 GUI)
  - 파일: `helpers/Sources/DashboardApp/main.swift`
  - 검증: `break-reminder dashboard --gui` → 네이티브 윈도우

- [ ] **P3-06** config 커맨드 (show/edit/path)
  - 파일: `cmd/break-reminder/config_cmd.go`
  - 검증: `break-reminder config show` → YAML 출력

---

## Phase 4: 가이드 휴식

- [ ] **P4-01** 눈 운동 (20-20-20 규칙)
  - 파일: `internal/breakactivity/eye.go`
  - 검증: `break-reminder break eye` → 단계별 가이드 + 타이머

- [ ] **P4-02** 스트레칭 가이드
  - 파일: `internal/breakactivity/stretch.go`
  - 검증: `break-reminder break stretch` → 단계별 안내

- [ ] **P4-03** 호흡 운동 (박스 호흡)
  - 파일: `internal/breakactivity/breathe.go`
  - 검증: `break-reminder break breathe` → 4초 사이클 애니메이션

- [ ] **P4-04** 산책 타이머
  - 파일: `internal/breakactivity/walk.go`
  - 검증: `break-reminder break walk` → 카운트다운

---

## Phase 5: AI 연동

- [ ] **P5-01** AI 클라이언트 (CLI 래퍼)
  - 파일: `internal/ai/client.go`
  - 검증: `ai.Available()` → claude CLI 존재 여부

- [ ] **P5-02** 히스토리 파일 관리
  - 파일: `internal/ai/history.go`
  - 검증: `AppendHistory()` → JSON 파일 append + `LoadHistory()` 라운드트립

- [ ] **P5-03** ai summary 커맨드
  - 파일: `cmd/break-reminder/ai.go`
  - 검증: `break-reminder ai summary` → AI 생산성 리포트

- [ ] **P5-04** ai suggest 커맨드
  - 검증: `break-reminder ai suggest` → 최적 타이밍 제안

- [ ] **P5-05** ai configure 커맨드
  - 검증: `break-reminder ai configure "25분 작업, 5분 휴식"` → diff 표시 + 확인

---

## Phase 6: 배포

- [ ] **P6-01** VERSION 파일 + Makefile ldflags
  - 파일: `VERSION`, `Makefile`
  - 검증: `break-reminder version` → VERSION 파일 내용 출력

- [ ] **P6-02** GitHub Actions CI
  - 파일: `.github/workflows/ci.yml`
  - 검증: push 시 빌드+테스트 통과

- [ ] **P6-03** GitHub Actions Release
  - 파일: `.github/workflows/release.yml`
  - 검증: 태그 push → GitHub Release 생성

- [ ] **P6-04** Homebrew Formula + Tap
  - 파일: `Formula/break-reminder.rb`
  - 검증: `brew install devlikebear/tap/break-reminder`

---

## 완료 기준

모든 체크리스트 항목이 완료되면:
1. `make build` → Go + Swift 바이너리 빌드 성공
2. `go test ./...` → 모든 단위 테스트 통과
3. `swift test --package-path helpers/` → Swift 테스트 통과
4. `break-reminder doctor` → 모든 진단 통과
5. `brew install devlikebear/tap/break-reminder` → 설치 성공
