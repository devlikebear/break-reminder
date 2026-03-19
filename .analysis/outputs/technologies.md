# 기술 스택 상세

## Go 패키지

| 패키지 | 용도 | 사용 위치 |
|--------|------|-----------|
| `github.com/spf13/cobra` | CLI 프레임워크 (서브커맨드, 플래그) | `cmd/break-reminder/*.go` |
| `github.com/charmbracelet/bubbletea` | TUI 프레임워크 (Elm 아키텍처) | `internal/dashboard/` |
| `github.com/charmbracelet/lipgloss` | TUI 스타일링 (색상, 레이아웃) | `internal/dashboard/` |
| `github.com/charmbracelet/bubbles` | TUI 컴포넌트 (key binding) | `internal/dashboard/` |
| `github.com/rs/zerolog` | 구조화 로깅 | `cmd/break-reminder/main.go` |
| `gopkg.in/yaml.v3` | YAML 파싱/직렬화 | `internal/config/` |

## Swift / macOS

| 기술 | 용도 | 사용 위치 |
|------|------|-----------|
| Swift Package Manager | 빌드 시스템 | `helpers/Package.swift` |
| AppKit (NSWindow, NSView) | 네이티브 UI | `BreakScreenApp`, `DashboardApp` |
| `NSWindow.Level.screenSaver` | 풀스크린 잠금 | `BreakScreenApp` |
| `NSBezierPath` | 원형 프로그레스 바 | `DashboardApp` |
| `Process` | 시스템 명령 실행 | `DashboardApp` (launchctl, ioreg) |

## macOS 시스템 연동

| 기능 | 구현 방식 |
|------|-----------|
| 유휴 감지 | `ioreg -c IOHIDSystem` → HIDIdleTime 파싱 |
| 알림 | `osascript -e 'display notification'` |
| TTS | `say -v <voice> "<text>"` |
| 서비스 | LaunchAgent plist + `launchctl load/unload` |
| 선택 다이얼로그 | `osascript -e 'choose from list'` |

## 빌드 & 배포

| 도구 | 역할 |
|------|------|
| Makefile | 로컬 빌드 (`make build`), 설치, 릴리스 아카이브 |
| GitHub Actions CI | push/PR 시 빌드 + 테스트 |
| GitHub Actions Release | 태그 push 시 바이너리 빌드 → GitHub Release → Homebrew tap 업데이트 |
| Homebrew Formula | `brew install devlikebear/tap/break-reminder` |
