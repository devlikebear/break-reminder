# 모듈: Cobra 커맨드 (cmd/break-reminder/)

## 커맨드 트리

```
break-reminder
├── check          # launchd가 60초마다 호출하는 단일 tick
├── status         # 현재 상태 출력 (모드, 작업시간, 유휴)
├── dashboard      # TUI 대시보드 (--gui로 네이티브 GUI)
├── daemon         # 포그라운드 루프 (Ctrl+C로 종료)
├── reset          # 타이머 초기화
├── doctor         # 시스템 진단 (TTS, 알림, 유휴감지 등)
├── break          # 가이드 휴식 활동
│   └── [eye|stretch|breathe|walk]
├── config
│   ├── show       # 현재 설정 YAML 출력
│   ├── edit       # $EDITOR로 설정 파일 열기
│   └── path       # 설정 파일 경로 출력
├── ai
│   ├── suggest    # AI 기반 최적 타이밍 분석
│   ├── summary    # AI 기반 생산성 리포트 (--weekly)
│   └── configure  # 자연어로 설정 변경
├── service
│   ├── install    # LaunchAgent 등록
│   ├── uninstall  # LaunchAgent 제거
│   ├── start/stop/status
└── version        # 버전 출력
```

## 주요 파일별 역할

| 파일 | 역할 |
|------|------|
| `main.go` | Cobra root 커맨드, PersistentPreRunE로 config 로드 |
| `check.go` | `runCheck()` — 타이머 tick 실행 + 액션 처리 |
| `daemon.go` | `time.Ticker`로 check 반복 + SIGINT/SIGTERM 처리 |
| `dashboard.go` | TUI 실행 또는 `--gui` 시 Swift DashboardApp 헬퍼 실행 |
| `break.go` | 가이드 활동 선택 → Bubbletea Model 실행 |
| `ai.go` | AI CLI 호출 (suggest/summary/configure) |
| `service.go` | launchd 관리 (install/uninstall/start/stop/status) |
| `config_cmd.go` | 설정 조회/편집 |
| `doctor.go` | 진단 실행 + 결과 포매팅 |
| `status.go` | 상태 한 눈에 보기 + `fmtMin()` 포매터 |
