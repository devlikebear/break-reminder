# 모듈: timer

**경로**: `internal/timer/timer.go`
**역할**: 타이머 핵심 로직. 순수 함수로 구현되어 테스트가 용이.

## 핵심 함수

### `Tick(cfg, state, now, idleSec) → TickResult`
매 체크 주기(60초)마다 호출. 현재 상태를 입력받아 다음 상태와 실행할 액션 목록을 반환.

**처리 순서**:
1. **일간 리셋** — 날짜가 바뀌면 `TodayWorkSeconds`/`TodayBreakSeconds` 초기화. 이전 날 데이터는 `DayEndSummary`에 보관.
2. **장기 부재 감지** — `elapsed > 3600s`면 리셋 (컴퓨터 재시작 등).
3. **중간 갭 감지** — `elapsed > checkInterval × 3`이면 슬립으로 간주, 시간 무시.
4. **tickWork** — 유휴 미만이면 작업 시간 누적. 50분 도달 → 휴식 전환 + 알림.
5. **tickBreak** — 휴식 경과 계산. 10분 도달 → 작업 전환 + 알림.

## Action 열거형

| Action | 트리거 조건 |
|--------|-------------|
| `ActionNotifyBreakTime` | 작업 50분 도달 |
| `ActionNotifyBreakOver` | 휴식 10분 완료 |
| `ActionNotifyFiveMinWarning` | 작업 45분 (5분 전 경고) |
| `ActionNotifyStillOnBreak` | 휴식 중 사용자 활동 감지 |
| `ActionSpeakBreakTime/Over` | TTS 활성화 시 음성 안내 |
| `ActionSaveDailyHistory` | 일간 리셋 시 히스토리 저장 |

## 테스트
`internal/timer/timer_test.go` — 테이블 드리븐 테스트로 모든 모드 전환 시나리오 커버.
