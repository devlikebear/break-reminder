# Break Reminder — 프로젝트 개요

## 한 줄 요약
macOS 전용 작업/휴식 사이클 관리 도구. 50분 작업 → 10분 휴식을 반복하며, 풀스크린 잠금 화면·TUI 대시보드·AI 분석·가이드 휴식 활동을 제공한다.

## 핵심 기술 스택
| 레이어 | 기술 |
|--------|------|
| 메인 바이너리 | Go 1.24 + Cobra CLI |
| TUI 대시보드 | Bubbletea + Lipgloss |
| 네이티브 UI (잠금화면·GUI 대시보드) | Swift + AppKit (SPM) |
| 스케줄링 | macOS LaunchAgent (launchd) |
| 설정 | YAML (`~/.config/break-reminder/config.yaml`) |
| 상태 저장 | 플랫 key=value 파일 (`~/.break-reminder-state`) |
| AI 연동 | Claude CLI / Codex CLI 서브프로세스 |
| 배포 | Homebrew tap + GitHub Actions |

## 실행 모드
1. **check** — launchd가 60초마다 호출. 타이머 tick 1회 실행.
2. **daemon** — 포그라운드 루프. check와 동일하지만 터미널에서 직접 실행.
3. **dashboard** — Bubbletea TUI 또는 `--gui` 플래그로 네이티브 GUI.
4. **break** — 가이드 휴식 활동 (눈운동·스트레칭·호흡·산책).
5. **ai** — AI 기반 분석/요약/설정 변경.

## 저장소 구조 (요약)
```
cmd/break-reminder/    # Cobra 커맨드 (main, check, daemon, dashboard, ai, ...)
internal/              # Go 내부 패키지 (timer, config, state, idle, notify, tts, ...)
helpers/               # Swift SPM 패키지 (BreakScreenApp, DashboardApp, HelperCore)
Formula/               # Homebrew formula
.github/workflows/     # CI + Release
```

## 버전
`VERSION` 파일이 단일 소스 오브 트루스. 현재 `0.1.1`.
