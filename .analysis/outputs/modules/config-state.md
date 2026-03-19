# 모듈: config & state

## config (`internal/config/`)

**파일**: `types.go`, `defaults.go`, `load.go`

YAML 파일(`~/.config/break-reminder/config.yaml`)에서 설정을 읽고, 기본값과 병합.

### 주요 설정 필드
- `work_duration_min` (50) / `break_duration_min` (10) — 작업/휴식 시간
- `idle_threshold_sec` (120) — 유휴 판단 기준
- `natural_break_sec` (300) — 자연 휴식 감지
- `work_days` / `work_start_hour` / `work_end_hour` — 근무 스케줄
- `break_screen_mode` ("ask"/"block"/"notify") — 잠금 화면 모드
- `ai_enabled` / `ai_cli` — AI 기능

### Boolean 머지 전략
Go에서 `bool`의 zero value가 `false`이므로 YAML에 명시적으로 `false`를 쓴 것과 아예 안 쓴 것을 구분할 수 없음. `raw map[string]any`로 키 존재 여부를 확인하여 해결.

---

## state (`internal/state/`)

**파일**: `state.go`

`~/.break-reminder-state`에 key=value 형식으로 저장. 기존 bash 스크립트 호환.

### State 필드
| 필드 | 의미 |
|------|------|
| `WorkSeconds` | 현재 세션 작업 누적 (초) |
| `Mode` | "work" 또는 "break" |
| `LastCheck` | 마지막 체크 유닉스 타임스탬프 |
| `BreakStart` | 현재 휴식 시작 시각 |
| `TodayWorkSeconds` | 오늘 총 작업 시간 |
| `TodayBreakSeconds` | 오늘 총 휴식 시간 |
| `LastUpdateDate` | 마지막 날짜 (YYYY-MM-DD) |

### 파일 형식 예시
```
WORK_SECONDS=601
MODE=work
LAST_CHECK=1773900414
BREAK_START=1773891945
TODAY_WORK_SECONDS=3600
TODAY_BREAK_SECONDS=600
LAST_UPDATE_DATE=2026-03-19
```
