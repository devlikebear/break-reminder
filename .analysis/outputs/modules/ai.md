# 모듈: AI 연동

**경로**: `internal/ai/`
**역할**: 외부 AI CLI(Claude, Codex)를 서브프로세스로 호출하여 생산성 분석, 설정 변경을 지원.

## 파일 구조

| 파일 | 역할 |
|------|------|
| `client.go` | AI CLI 래퍼 (Query, Available) |
| `history.go` | 일간 히스토리 파일 관리 (Load, Append) |
| `ai_test.go` | 히스토리 라운드트립 테스트 |

## Client (`client.go`)

```go
type Client struct {
    CLIName string        // "claude" 또는 "codex"
    Timeout time.Duration // 120초 (기본)
}
```

- **`Available()`** — `exec.LookPath`로 CLI 존재 여부 확인
- **`Query(ctx, prompt)`** — CLI별 분기:
  - claude: `claude -p <prompt> --output-format text --max-turns 1`
  - codex: `codex -q <prompt>`
  - 120초 타임아웃 (이전에 30초로 인해 `signal: killed` 오류 발생 → 수정)

## History (`history.go`)

```go
type DailySummary struct {
    Date       string `json:"date"`
    WorkMin    int    `json:"work_min"`
    BreakMin   int    `json:"break_min"`
    Sessions   int    `json:"sessions"`
    Activities int    `json:"activities"`
}
```

- **파일 경로**: `~/.break-reminder-history.json`
- **`LoadHistory()`** — JSON 배열 로드 (파일 없으면 nil 반환)
- **`AppendHistory(summary)`** — 같은 날짜가 있으면 업데이트, 없으면 추가

### 데이터 흐름

```
timer.Tick() 일간 리셋
  → ActionSaveDailyHistory + DayEndSummary
  → check.go executeActions()
  → ai.AppendHistory()
  → ~/.break-reminder-history.json
```

`ai summary` 실행 시:
```
history.json (과거 데이터) + state 파일 (오늘 실시간 보간)
  → 프롬프트 조합 → Client.Query() → AI 응답 출력
```

## 커맨드 (`cmd/break-reminder/ai.go`)

| 서브커맨드 | 동작 |
|------------|------|
| `ai summary` | 오늘 + 과거 히스토리 기반 생산성 리포트 |
| `ai summary --weekly` | 최근 7일 데이터 분석 |
| `ai suggest` | 최적 작업/휴식 타이밍 제안 |
| `ai configure "<자연어>"` | 자연어 → config 변경 diff → 확인 후 적용 |

## 테스트

`ai_test.go` — `historyPathOverride`로 임시 파일 사용, AppendHistory + LoadHistory 라운드트립 검증.
