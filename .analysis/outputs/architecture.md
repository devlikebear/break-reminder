# 아키텍처

## 전체 구조

```
┌──────────────────────────────────────────────────────┐
│  macOS LaunchAgent (60s 간격) 또는 daemon 루프        │
│    ↓ break-reminder check                            │
├──────────────────────────────────────────────────────┤
│  cmd/break-reminder/check.go                         │
│    1. state.Load()          — 상태 파일 읽기          │
│    2. idle.IdleSeconds()    — 시스템 유휴 시간         │
│    3. timer.Tick(cfg,s,now) — 순수 함수 타이머 로직    │
│    4. executeActions()      — 알림/TTS/잠금화면 실행   │
│    5. state.Save()          — 상태 파일 저장           │
├──────────────────────────────────────────────────────┤
│  timer.Tick() 로직 (순수 함수, 부수효과 없음)          │
│    ┌─ 일간 리셋 (날짜 변경 감지 → 히스토리 저장)       │
│    ├─ 갭 감지 (슬립/재시작 → 시간 무시)               │
│    ├─ tickWork: 작업 시간 누적 → 50분 도달 시 휴식     │
│    └─ tickBreak: 휴식 완료 감지 → 작업 모드 복귀       │
├──────────────────────────────────────────────────────┤
│  플랫폼 추상화 (interface + build tag)                │
│    idle.Detector   — darwin: ioreg / stub: 0         │
│    notify.Notifier — darwin: osascript / stub: noop  │
│    tts.Speaker     — darwin: say / stub: noop        │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│  Swift 네이티브 헬퍼 (서브프로세스로 실행)             │
│    BreakScreenApp — 풀스크린 잠금 화면 (멀티모니터)    │
│    DashboardApp   — 네이티브 GUI 대시보드              │
│    HelperCore     — 공유 순수 로직 (파싱, 진행률 계산)  │
└──────────────────────────────────────────────────────┘
```

## 데이터 흐름

### 타이머 사이클
```
[launchd 60s] → check.go → timer.Tick()
                              ↓
                    TickResult { State, Actions, DayEndSummary }
                              ↓
                    executeActions() → notify / tts / breakscreen / history
                              ↓
                    state.Save() → ~/.break-reminder-state
```

### 설정 로드
```
~/.config/break-reminder/config.yaml
        ↓ yaml.Unmarshal
    raw map (boolean 감지용) + typed struct
        ↓ merge(dst, src, raw)
    Config (기본값 + YAML 오버라이드)
```

### AI 연동
```
ai summary/suggest → ai.LoadHistory() + state.Load()
    → 프롬프트 생성 → claude -p "..." --output-format text
    → 응답 출력
```

## 핵심 설계 결정

1. **timer.Tick()은 순수 함수** — 입력(cfg, state, now, idleSec)만 받고, 출력(TickResult)만 반환. 부수효과는 호출자(check.go)가 처리. 테스트 용이.

2. **플랫폼 분리는 build tag** — `//go:build darwin`과 `//go:build !darwin`으로 macOS 전용 코드를 격리. interface로 추상화.

3. **Swift 헬퍼는 서브프로세스** — Go에서 `exec.Command`로 실행. 프로세스 간 통신은 CLI 인자로만.

4. **상태 파일은 key=value** — 기존 bash 스크립트와 호환. JSON이 아닌 이유는 하위 호환성.

5. **Boolean 머지는 raw map** — YAML에서 `false`를 명시적으로 설정한 것과 기본값 `false`를 구분하기 위해 `map[string]any`로 키 존재 여부를 확인.

## 테스트 구조

```
internal/timer/timer_test.go       # 테이블 드리븐 — 모든 모드 전환 + 갭 감지
internal/config/config_test.go     # YAML 로드 + boolean 머지
internal/state/state_test.go       # key=value 라운드트립
internal/ai/ai_test.go            # 히스토리 append + load
internal/doctor/doctor_test.go     # 진단 실행 확인
internal/schedule/schedule_test.go # 근무시간/요일 판정
internal/logging/logging_test.go   # 로그 + 로테이션
internal/breakscreen/breakscreen_test.go  # 오케스트레이션 단위 테스트
helpers/Tests/HelperCoreTests/     # Swift 테스트 (5파일, 35+ 케이스)
  ArgsParserTests.swift            # CLI 인자 파싱 + formatMinutes
  ConfigParserTests.swift          # YAML 간이 파서
  StateParserTests.swift           # state 파일 파서
  ProgressCalcTests.swift          # 진행률 계산
  TimeFormatterTests.swift         # 초→"mm:ss" 포맷
```

**원칙**: timer.Tick()이 순수 함수이므로 핵심 로직의 단위 테스트가 외부 의존 없이 가능. Swift도 HelperCore에 순수 로직을 분리하여 테스트 가능하게 설계.

## 스크립트 & 배포 아티팩트

| 파일 | 역할 |
|------|------|
| `install.sh` | 소스 빌드 + `~/.local/bin/` 설치 + LaunchAgent 등록 (원클릭) |
| `uninstall.sh` | LaunchAgent 해제 + 바이너리/상태파일 삭제 |
| `break-reminder.sh` | 기존 bash 버전 (레거시, 참조용) |
| `com.user.break-reminder.plist` | 예시 LaunchAgent plist 템플릿 |
| `config/default.yaml` | 기본 설정 파일 레퍼런스 |
| `VERSION` | 단일 소스 오브 트루스 (현재 0.1.1) |
| `Formula/break-reminder.rb` | Homebrew formula (릴리스 시 자동 업데이트) |
