# 업무 리듬 비서 리뉴얼 — 개발 로드맵

작성일: 2026-10-20 (rev.2: tars `pkg/llm` 도입, Mac App Store 대비) · 기준 HEAD: `cb33db7` (v0.16.2)

상태: **구현 전 개발계획**. 체크박스는 구현 완료를 뜻하지 않는다. 각 페이즈 Checkpoint에서 사용자 확인 후 다음으로 진행한다.

## 1. 제품 방향

Break Reminder를 휴식 알림 도구에서 **"내 업무 리듬을 아는 맥 메뉴바 비서"**로 확장한다. 캘린더·앱 사용·날씨·맥 리소스 신호를 기존 활동 감지/세션/휴식 데이터와 **결합**해 타이밍 있는 제안을 하고, AI는 브리핑·회고·질문에 답한다.

정보 표시만으로는 Stats, Itsycal, Raycast, macOS 위젯과 차별점이 없다. 모든 기능은 "기존 업무 리듬 데이터와 무엇을 결합하는가"로 정당화한다.

## 2. 확정된 결정

| 항목 | 결정 | 근거 |
|---|---|---|
| AI 런타임 | **tars `pkg/llm`** (linetta와 같은 방식) | 프로바이더 하나의 인터페이스로 `claude-code-cli`(구독)와 API 키(BYOK)를 모두 지원. linetta가 같은 구성으로 Mac App Store 심사 통과 |
| 기본 프로바이더 | 직접 배포판: `claude-code-cli` / 앱스토어판: Anthropic·OpenAI·Gemini 키, (선택) Codex 로그인 | 샌드박스에서 `claude` 실행은 EPERM (linetta `2026-06-17-macos-app-sandbox-mas-design.md`) |
| 툴 레이어 | 자체 MCP 서버 **하나** (`modelcontextprotocol/go-sdk`). 내장 에이전트는 인프로세스(`NewInMemoryTransports`), `claude-code-cli`와 외부 클라이언트는 127.0.0.1 HTTP + stdio 브리지 | linetta `2026-09-02-builtin-agent-byok-design.md` 4.1절 패턴. 툴을 고치면 내장 에이전트와 Claude Code가 함께 좋아짐 |
| 에이전트 루프 | 직접 작성한 작은 루프. `pkg/agentloop`·`pkg/session`은 쓰지 않음 | linetta와 동일 판단: 두 패키지는 tars에서 Experimental |
| 앱 구조 | Swift 메뉴바 `.app` + 번들 내장 Go **엔진 사이드카**(stdio JSON-RPC). launchd 60초 `check` 구조는 P4에서 퇴역 | 앱스토어 샌드박스는 `~/Library/LaunchAgents` 쓰기, `ioreg`/`say`/`terminal-notifier` 실행을 막음 |
| 캘린더 | macOS EventKit (Swift) | OAuth 불필요, 샌드박스에서 엔타이틀먼트로 허용 |
| 배포 | Developer ID 서명·공증 직접 배포(Homebrew cask) 먼저, 앱스토어는 P5에서 | 사용자 결정 ("잘 만들면 앱스토어") |
| 판단 vs AI | 판단(회의 중인가, 집중 가능 시간 등)은 규칙 기반 순수 함수, AI는 문장 생성·요약·질의응답만 | 테스트 가능성, 사용량 절약 |

## 3. 아키텍처

```
[Swift 메뉴바 앱 (.app, SMAppService 로그인 항목)]
  ├ 팝오버(SwiftUI), 휴식 오버레이, UNUserNotification, AVSpeechSynthesizer
  ├ 수집기: EventKit 일정, NSWorkspace 전면 앱, 유휴 시간
  └ stdio JSON-RPC ─▶ [Go 엔진 사이드카 (Contents/MacOS/engine)]
                        ├ 기존 timer/session/time-tools/schedule 로직 (Tick 루프)
                        ├ workcontext · weather · sysinfo (규칙 기반 판단)
                        ├ internal/provider (tars pkg/llm) + internal/agent (루프)
                        └ MCP 서버 ─ 인프로세스: 내장 에이전트
                                    └ 127.0.0.1 HTTP: claude-code-cli, Claude Desktop/Code(stdio 브리지)
[CLI (직접 배포판만)] ── 엔진 HTTP RPC로 status/pause/timer 등 호출
```

- 엔진은 OS 호출(알림, 음성, 오버레이, 유휴 감지)을 직접 하지 않는다. "해야 할 일"을 이벤트로 앱에 보내고 Swift가 실행한다. 그래서 같은 엔진이 샌드박스 안팎에서 동일하게 동작한다.
- 데이터 경로는 `internal/paths` 한 곳에서 결정한다(직접 배포판은 기존 `~/.config/break-reminder`, 앱스토어판은 컨테이너).
- 비밀값(API 키)은 Keychain. 프로바이더별 데이터 전송 동의를 따로 받는다(linetta v0.9.3 교훈).
- 사용자 문자열은 `internal/i18n/catalog.json` + `python3 scripts/generate-localizations.py`.

## 4. 배포판별 기능

| 기능 | 직접 배포판 | 앱스토어판 |
|---|---|---|
| 휴식·뽀모도로·타이머·통계 | ✓ | ✓ |
| 캘린더·앱 사용·회의 인식 휴식 | ✓ | ✓ |
| AI: claude-code-cli (구독) | ✓ | ✗ |
| AI: API 키 / Codex 로그인 | ✓ | ✓ |
| 외부 MCP (Claude Desktop/Code 연결) | ✓ | ✓ (localhost 서버 엔타이틀먼트) |
| 터미널 CLI | ✓ | ✗ |
| Claude Code 훅 연동 | ✓ | ✗ |
| 개발 캐시 정리 후보 | 자동 탐색 | 사용자가 폴더를 한 번 선택 |
| KittenTTS/Supertonic (Python) | ✓ | ✗ (시스템 음성·Gemini TTS) |

빌드 태그 `mas`로 갈라지는 지점을 최소화한다(linetta와 동일하게 팩토리에서 거부하는 방식).

## 5. 페이즈

| 페이즈 | 사용자에게 완결되는 결과 | 예상 공수 |
|---|---|---|
| [P0 기반](renewal-phase-0-foundation.md) | 서명된 `.app`의 팝오버에서 기존 휴식·타이머가 엔진 사이드카로 동작하고, 샌드박스 스파이크 결과가 문서로 남음 | 32–48h |
| [P1 업무 맥락](renewal-phase-1-work-context.md) | 다음 일정·집중 가능 시간, 회의 중 휴식 자동 보류, 앱별 사용 시간 | 24–36h |
| [P2 AI 비서](renewal-phase-2-ai-assistant.md) | tars 기반 에이전트: 출근 브리핑, 퇴근 회고, 팝오버 질문, 외부 MCP | 28–40h |
| [P3 환경 신호](renewal-phase-3-signals.md) | 날씨/미세먼지 휴식 추천, "왜 느리지?" 진단, Claude Code 훅(직접판) | 20–30h |
| [P4 리브랜딩·직접 배포](renewal-phase-4-rebrand.md) | 새 이름의 공증된 `.app`을 Homebrew cask로 배포, v0.16 사용자 자동 이전, launchd 구조 퇴역 | 20–30h |
| [P5 앱스토어](renewal-phase-5-app-store.md) | 샌드박스 빌드를 Mac App Store에 제출 | 16–24h |

의존: `P0 → P1 → P2 → (P3, P4) → P5`. P1 이후 1주 실사용으로 회의 인식 휴식 가치를 확인한다. P5는 직접 배포판을 2주 이상 써 보고 진행 여부를 정한다.

## 6. 범위 밖

- 클립보드 히스토리, 런처, 메모/할 일 (Raycast·Maccy와 차별점 없음)
- 일정 생성·수정, 프로세스 종료, 파일 삭제의 **자동 실행** (제안 + 사용자 버튼만)
- 창 제목·화면 내용 수집, 원격 텔레메트리, 클라우드 동기화
- iOS/iPad 버전

## 7. 참고 자료 (사용자 저장소)

- tars: `pkg/llm`, `pkg/mcp`, `docs/public-agent-packages.md` (안정성 정책: `pkg/llm` Stable)
- linetta: `engine/internal/provider`, `engine/internal/agent`, `engine/internal/codexauth`, `docs/superpowers/specs/2026-06-17-macos-app-sandbox-mas-design.md`, `2026-06-18-mac-app-store-submission-design.md`, `2026-09-02-builtin-agent-byok-design.md`

## 8. 열린 질문

- 새 이름 → P4 시작 시 결정 (앱스토어 이름 중복 확인 포함). 후보: Hammy, Pacer, Workmate.
- tars 버전 고정: linetta는 v0.37.1, tars 최신은 v0.61.x. P2 시작 시 최신 안정 `pkg/llm`으로 고정하고 linetta와 맞출지 결정.
