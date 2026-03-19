# 클론 코딩 가이드

이 문서는 Break Reminder를 처음부터 다시 구현하는 과정을 단계별로 안내합니다.

---

## Step 1: 프로젝트 스캐폴딩

### 목표
Go 모듈 초기화, 디렉토리 구조 생성, 기본 Cobra CLI 세팅.

### 작업
1. `go mod init github.com/devlikebear/break-reminder`
2. 디렉토리 생성:
   ```
   cmd/break-reminder/
   internal/config/
   internal/state/
   internal/timer/
   ```
3. `cmd/break-reminder/main.go` — Cobra root 커맨드 + `PersistentPreRunE`에서 config 로드
4. `Makefile` — `build`, `install`, `clean` 타겟

### 핵심 코드
- `cmd/break-reminder/main.go`: Cobra 루트 커맨드, 버전 플래그, config 로드

### 검증
```bash
go build ./cmd/break-reminder && ./bin/break-reminder --help
```

---

## Step 2: 설정 시스템

### 목표
YAML 파일 로드 + 기본값 병합. Boolean 필드의 zero-value 문제 해결.

### 작업
1. `internal/config/types.go` — `Config` struct 정의 (yaml 태그)
2. `internal/config/defaults.go` — `DefaultConfig()` 함수
3. `internal/config/load.go` — `Load(path)` 함수:
   - YAML → `map[string]any` (raw) + typed struct 두 번 언마샬
   - `merge(dst, src, raw)`: raw에 키가 있으면 src 값 사용, 없으면 dst(기본값) 유지
4. `internal/config/load.go` — `Save(path, cfg)` 함수 (ask→block 선택 결과 저장용)

### 핵심 패턴
```go
// Boolean 머지: raw map에 키가 있으면 명시적 설정으로 판단
if _, ok := raw["tts_enabled"]; ok {
    dst.TTSEnabled = src.TTSEnabled
}
```

### 검증
```bash
go test ./internal/config/
```

---

## Step 3: 상태 파일

### 목표
key=value 플랫 파일로 상태 저장/로드. 기존 bash 스크립트 호환.

### 작업
1. `internal/state/state.go` — `State` struct, `Load()`, `Save()` 함수
2. 포맷: `KEY=VALUE` (한 줄에 하나)
3. 필드 매핑: `WORK_SECONDS`, `MODE`, `LAST_CHECK`, `BREAK_START`, `TODAY_WORK_SECONDS`, `TODAY_BREAK_SECONDS`, `LAST_UPDATE_DATE`

### 검증
```bash
go test ./internal/state/
```

---

## Step 4: 타이머 핵심 로직

### 목표
순수 함수 `Tick()`으로 모든 타이머 로직 구현. 부수효과 없음.

### 작업
1. `internal/timer/timer.go`:
   - `Action` 열거형 (상수)
   - `TickResult` struct (State + Actions + LogMsg + DayEndSummary)
   - `Tick(cfg, state, now, idleSec)` 함수:
     a. 일간 리셋 (날짜 변경 → 히스토리 저장 액션)
     b. 장기 부재 감지 (1시간+)
     c. 중간 갭 감지 (checkInterval × 3)
     d. `tickWork()` — 유휴 판단 → 작업 시간 누적 → 50분 도달 시 휴식 전환
     e. `tickBreak()` — 휴식 경과 계산 → 10분 도달 시 작업 전환

2. `internal/timer/timer_test.go` — 테이블 드리븐 테스트

### 핵심 패턴
```go
// 순수 함수: 입력만 받고 결과만 반환
func Tick(cfg config.Config, s state.State, now time.Time, idleSec int) TickResult {
    result := TickResult{State: s}
    // ... 로직 ...
    return result
}
```

### 검증
```bash
go test -v ./internal/timer/
```

---

## Step 5: 플랫폼 추상화

### 목표
macOS 전용 기능을 interface + build tag로 분리.

### 작업 (각각 3파일 패턴)
1. **idle** — `internal/idle/`
   - `idle.go`: `Detector` interface (`IdleSeconds() int`)
   - `idle_darwin.go`: `ioreg -c IOHIDSystem` → HIDIdleTime 파싱
   - `idle_stub.go`: 항상 0

2. **notify** — `internal/notify/`
   - `osascript display notification`

3. **tts** — `internal/tts/`
   - `say -v <voice> "<message>"`

### 핵심 패턴
```go
//go:build darwin

package idle

type darwinDetector struct{}

func New() Detector { return &darwinDetector{} }
```

### 검증
```bash
go build ./internal/idle/ && go build ./internal/notify/ && go build ./internal/tts/
```

---

## Step 6: check 커맨드

### 목표
launchd에서 60초마다 호출되는 메인 진입점.

### 작업
1. `cmd/break-reminder/check.go`:
   - `state.Load()` → `idle.IdleSeconds()` → `timer.Tick()` → `executeActions()` → `state.Save()`
2. `executeActions()` — Action별 분기:
   - notify/tts/breakscreen/history 처리

### 검증
```bash
./bin/break-reminder check
cat ~/.break-reminder-state
```

---

## Step 7: status / reset / daemon

### 목표
기본 유틸리티 커맨드 추가.

### 작업
1. `status.go` — 상태 파일 읽기 + 포매팅 출력 (`fmtMin()` 헬퍼)
2. `reset.go` — 상태 초기화
3. `daemon.go` — `time.Ticker`로 check 반복 + SIGINT/SIGTERM 핸들링

---

## Step 8: TUI 대시보드

### 목표
Bubbletea + Lipgloss로 실시간 대시보드.

### 작업
1. `internal/dashboard/dashboard.go` — Bubbletea Model:
   - `Init()`: 1초 tick 시작
   - `Update()`: 키 입력(q/r/b) + tick 처리
   - `View()`: 프로그레스 바 + 상태 + 통계 렌더링
2. `cmd/break-reminder/dashboard.go` — TUI 실행 또는 `--gui` 시 Swift 헬퍼

### 핵심 패턴
```go
// Elm 아키텍처: Model → Update → View
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg: // 키 입력 처리
    case tickMsg:    // 1초마다 상태 갱신
    }
}
```

---

## Step 9: 풀스크린 잠금 화면

### 목표
Swift AppKit 헬퍼로 풀스크린 블로킹 오버레이.

### 작업
1. `helpers/Package.swift` — SPM 패키지 정의
2. `helpers/Sources/HelperCore/` — 공유 순수 로직 (ArgsParser, TimeFormatter 등)
3. `helpers/Sources/BreakScreenApp/main.swift`:
   - `NSWindow.Level.screenSaver` borderless 윈도우
   - 멀티모니터: `NSScreen.screens` 순회
   - Skip 버튼 (2분 후 활성화), Esc 종료
4. `internal/breakscreen/` — Go에서 Swift 헬퍼 실행 오케스트레이션

### 검증
```bash
make build-helper
./bin/break-screen --duration 10
```

---

## Step 10: 가이드 휴식 활동

### 목표
눈운동, 스트레칭, 호흡, 산책 — 각각 Bubbletea Model.

### 작업
1. `internal/breakactivity/` — 4개 활동 모델
2. `cmd/break-reminder/break.go` — 서브커맨드 (eye/stretch/breathe/walk)

---

## Step 11: AI 연동

### 목표
Claude CLI를 통한 생산성 분석 + 자연어 설정 변경.

### 작업
1. `internal/ai/client.go` — CLI 래퍼 (120초 타임아웃, `--max-turns 1`)
2. `internal/ai/history.go` — `~/.break-reminder-history.json` 관리
3. `cmd/break-reminder/ai.go` — suggest/summary/configure 서브커맨드

---

## Step 12: 서비스 관리 + 배포

### 목표
LaunchAgent 등록 + Homebrew 배포 파이프라인.

### 작업
1. `internal/launchd/` — plist 생성 + launchctl 명령
2. `cmd/break-reminder/service.go` — install/uninstall/start/stop/status
3. `Formula/break-reminder.rb` — Homebrew formula
4. `.github/workflows/ci.yml` — push/PR 시 빌드+테스트
5. `.github/workflows/release.yml` — 태그 → 릴리스 → tap 업데이트
