# 모듈: TUI 대시보드 & 가이드 휴식

**경로**: `internal/dashboard/`
**역할**: Bubbletea TUI 대시보드 + 내장 가이드 휴식 활동 4종.

## 파일 구조

| 파일 | 역할 |
|------|------|
| `dashboard.go` | 메인 대시보드 Model (Elm 아키텍처) |
| `activities.go` | 4가지 가이드 휴식 활동 (Eye, Stretch, Breathe, Walk) |

## 대시보드 (`dashboard.go`)

### Model
```go
type Model struct {
    cfg              config.Config
    state            state.State
    idleSec          int
    logs             []string
    showBreakMenu    bool       // 휴식 활동 선택 메뉴
    breakActivity    tea.Model  // 현재 실행 중인 활동
    showingActivity  bool
}
```

### Elm 아키텍처 흐름
1. **Init**: 1초 tick 시작
2. **Update**:
   - `tickMsg`: state/config/idle/logs 새로 읽기 (1초마다)
   - `tea.KeyMsg`: q=종료, r=리셋, b=강제 휴식
   - 휴식 모드 진입 시 `break_activities_enabled: true`면 활동 선택 메뉴 표시
3. **View**: 시스템 상태 + 프로그레스 바 + 일간 통계 + 로그

### 화면 구성
```
🐹 Break Reminder Dashboard (q:quit r:reset b:break)
══════════════════════════════════════════════════
System: Installed & Running
Status: WORKING
Idle: 3s / Limit: 120s

Session Work: [██████████░░░░░░░░░░] 50%  (25 / 50 min)

Daily Statistics:
  Work: 2h 5m
  Rest: 30m
  Ratio: [████████████████░░░░] 80%

Recent Logs:
──────────────────────────────────────────────────
  [2026-03-19 10:00:00] work mode, session 25min
──────────────────────────────────────────────────
```

### 키 바인딩

| 키 | 동작 |
|----|------|
| `q` / `Ctrl+C` | 종료 |
| `r` | 타이머 리셋 |
| `b` | 강제 휴식 전환 |
| `j/k` / `↑↓` | 활동 메뉴 탐색 |
| `Enter` | 활동 선택 |
| `Esc` | 메뉴/활동 닫기 |

## 가이드 휴식 활동 (`activities.go`)

모두 `tea.Model` interface 구현. 대시보드 내에서 오버레이로 표시.

| 활동 | 구조체 | 시간 | 설명 |
|------|--------|------|------|
| 👁 눈 운동 | `EyeActivity` | 2분 | 20-20-20 규칙: 20초 × 3라운드 |
| 🤸 스트레칭 | `StretchActivity` | 5분 | 5단계 (목/어깨/손목/기립/자유) × 60초 |
| 🌬 호흡 | `BreatheActivity` | 4분 | 박스 호흡법: 4초 × 4단계 = 16초 사이클 × 15 |
| 🚶 산책 | `WalkActivity` | 5분 | 단순 카운트다운 타이머 |

### 공통 패턴
```go
type XxxActivity struct {
    startTime time.Time
    elapsed   time.Duration
    totalDur  time.Duration
}
```
- `Init()`: 1초 tick 시작
- `Update(tickMsg)`: elapsed 갱신, 완료 시 nil 반환
- `View()`: 단계별 진행 표시 + 남은 시간
- Esc로 중도 종료 (대시보드에서 처리)
