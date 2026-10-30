# Phase 2: AI 비서 (tars) — 작업지시서

_작성일: 2026-10-20 (rev.2) · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 28–40h_

## 페이즈 목표

tars `pkg/llm`으로 프로바이더를 붙이고, 앱의 맥락을 MCP 툴 하나의 레이어로 노출한다. 끝나면 첫 활동 시 햄스터가 오늘 브리핑을 음성으로 말하고, 세션 종료 시 회고를 남기며, 팝오버에서 자연어로 질문할 수 있다. 같은 툴을 Claude Code/Desktop에서도 쓸 수 있다.

## 전제 조건

- [x] P1 완료, P0 비밀값(Keychain) 완료
- [x] tars 버전 고정 결정 → v0.66.1 (최신 태그, `pkg/llm` Stable)

## 포함 기능

1. MCP 서버 (툴 레지스트리 하나, 인프로세스 + 127.0.0.1 HTTP + stdio 브리지)
2. 프로바이더 설정: claude-code-cli(직접판 기본), Anthropic·OpenAI·Gemini 키, 프로바이더별 동의
3. 내장 에이전트 루프
4. 출근 브리핑 / 퇴근 회고, 팝오버 질문창

## 이 페이즈에서 하지 않는 것

- 날씨·리소스 툴 → P3
- Codex(ChatGPT) 로그인 → 앱스토어판 준비 때(P5) linetta `internal/codexauth` 이식 검토
- 메모리·스킬 → 범위 밖 (필요 시 tars `pkg/memory`, `pkg/skill` 검토)
- 위험 동작 자동 실행 → 범위 밖

## 작업 체크리스트

### A. MCP 서버

- [x] **T2.A.1** — 툴 정의
  - 파일: `internal/mcphost/tools.go`, 테스트
  - 내용: go-sdk 서버에 등록. 읽기 툴 `get_status`, `get_today_calendar`, `get_focus_window`, `get_app_usage(days)`, `get_work_history(days)`. 동작 툴(되돌릴 수 있는 것만) `start_pomodoro`, `start_timer(duration,label)`, `pause(mode,duration)`, `resume`. 툴 설명에 사용 흐름을 적어 모델에 그대로 전달(linetta 방식). 일정 제목 전달은 설정 `share_event_titles`.
  - 검증: 가짜 엔진 상태로 툴 단위 테스트
- [x] **T2.A.2** — 전송 3종
  - 파일: `internal/mcphost/{inprocess.go,http.go}`, `cmd/break-reminder/mcp.go`(stdio 브리지)
  - 내용: 내장 에이전트용 `mcp.NewInMemoryTransports()`, 외부용 127.0.0.1 HTTP(랜덤 포트 + 토큰, 포트·토큰은 `paths` 아래 discovery 파일), stdio 브리지는 discovery 파일을 읽어 HTTP로 중계.
  - 참조: linetta `engine/internal/mcphost`, `engine/cmd/linetta-mcp`
  - 검증: initialize → tools/list → tools/call 왕복 테스트, 수동으로 `claude mcp add` 후 Claude Code에서 "내 다음 일정"

### B. 프로바이더와 에이전트

- [x] **T2.B.1** — `internal/provider`
  - 내용: 설정 → `llm.ProviderOptions` → `llm.NewProvider`. 모델 목록 조회, 연결 테스트. `//go:build mas`에서 `claude-code-cli`는 팩토리에서 거부. 기존 `ai_cli: claude` 설정은 `claude-code-cli`로 자동 매핑, `codex` CLI는 지원 종료 안내.
  - 의존성 규칙: `pkg/llm`은 `internal/provider`·`internal/agent`에서만 import. `go list -deps`로 확인하는 스크립트를 `make test`에 추가 (linetta `scripts/validate-story-core-deps.sh` 참조)
  - 검증: 가짜 `ClientFactory`로 테스트 (linetta `internal/agenttest/scripted_client.go` 패턴)
- [x] **T2.B.2** — `internal/agent` 루프
  - 내용: API 프로바이더: `ListTools` → `llm.ToolSchema` 변환 → `Chat` → 툴 호출을 인프로세스 MCP로 실행 → 반복(최대 8회, 같은 호출 반복 감지). `claude-code-cli`: Claude Code가 자체 루프를 돌리므로 HTTP MCP 엔드포인트를 MCP 설정으로 넘기고 허용 툴은 `mcp__<server>__*` 읽기 툴로 제한, 동작 툴은 사용자가 질문창에서 명시한 경우에만 허용.
  - 스트리밍: 엔진 `agent.delta` 이벤트로 앱에 전달
  - 검증: scripted client로 툴 호출 → 결과 → 최종 답 흐름 테스트
- [x] **T2.B.3** — 기존 `internal/ai`·`internal/insights` 흡수: 인사이트 생성을 `internal/agent` 단발 호출로 교체(TIDY-005 해소), `ai summary/suggest` CLI는 유지
- [x] **T2.B.4** — 설정 UI: 프로바이더 선택, 키 입력(Keychain), 모델, 연결 테스트, 프로바이더별 "일정·앱 사용 정보 전송" 동의

### C. 브리핑·회고

- [x] **T2.C.1** — `internal/assistant/briefing.go`
  - 내용: `session_start` 이벤트에서 하루 1회 브리핑, `session_end`에서 회고 생성. 프롬프트: 3문장 이내, 숫자 위주, 오늘 할 일 1개 제안, UI 언어 사용. 결과를 `paths` 아래 일별 파일에 저장하고 회고는 `ai.DailySummary`에 `retro` 필드로 추가. AI 불가·미동의 시 규칙 기반 템플릿 문장.
  - 참조: `internal/insights/prompt.go`의 언어 선택
- [x] **T2.C.2** — 전달: 알림 + 음성(설정 `briefing_voice`, 회의 중이면 생략) + 팝오버 상단 카드(닫을 때까지 유지)

### D. 질문창

- [x] **T2.D.1** — 팝오버 입력창 → 엔진 `agent.ask` RPC, 스트리밍 표시, 팝오버가 열려 있는 동안 대화 유지. 모델이 제안한 동작은 응답 아래 버튼으로 표시하고 누르면 해당 RPC 실행.

---

## ✅ Phase 2 Checkpoint

**자동:** `make test`(의존성 규칙 포함), MCP 왕복, scripted client 루프 테스트, `go build -tags mas ./...`

**수동:**
- [ ] claude-code-cli 프로바이더로 아침 첫 활동 → 알림 + 음성 브리핑, 같은 날 반복 없음
- [ ] Anthropic 키로 바꿔도 같은 질문에 툴을 호출해 답함
- [ ] 동의하지 않은 프로바이더는 템플릿 브리핑으로 대체
- [ ] Claude Code에 MCP 등록 후 툴 호출 성공
- [ ] 하루 AI 호출이 브리핑 1 + 회고 1 + 질문 수를 넘지 않음

**완료 처리:** 브리핑 말투·길이를 사용자와 조정하고 P3 또는 P4로 진행.
