# Phase 5: Mac App Store — 작업지시서

_작성일: 2026-10-20 · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 16–24h_

## 페이즈 목표

직접 배포판과 같은 코드에서 샌드박스 빌드를 만들어 Mac App Store에 제출한다. linetta가 거친 절차를 그대로 따른다.

## 진행 조건 (go/no-go)

- [ ] 직접 배포판을 2주 이상 사용, 데이터 유실·알림 누락 같은 미해결 문제 없음
- [ ] P0 스파이크의 FAIL 항목이 모두 대안으로 해결됨 (프로세스별 진단 → P3에서 직접판 전용 처리)
- [ ] 사용자가 진행 결정

## 작업 체크리스트

### A. 샌드박스 빌드

- [ ] **T5.A.1** — `make app-mas`: `-tags mas` 엔진, `app-sandbox` + 필요한 엔타이틀먼트만(캘린더, 네트워크 클라이언트, localhost 서버, 사용자 선택 파일, Keychain). 사이드카는 inherit 엔타이틀먼트
  - 참조: linetta `docs/superpowers/specs/2026-06-17-macos-app-sandbox-mas-design.md`
- [ ] **T5.A.2** — 직접판 전용 기능이 UI에서 숨겨지는지 확인 (claude-code-cli, CLI, Claude Code 훅, Python TTS)
- [ ] **T5.A.3** — Codex(ChatGPT) 로그인 지원 여부 결정, 지원 시 linetta `internal/codexauth` 이식(또는 tars 공개 패키지로 추출 제안)
- [ ] **T5.A.4** — 직접판 → 앱스토어판 데이터 가져오기: 자동 이전은 하지 않음(샌드박스가 `~/.config/break-reminder`·`~/.break-reminder-state` 읽기와 LaunchAgents 정리를 막음). 사용자가 P4 T4.C.4 내보내기 파일을 `NSOpenPanel`로 선택해 가져오고, 기존 launchd 서비스 정리는 직접판 `<cli> service uninstall` 안내로 대신함

### B. 제출

- [ ] **T5.B.1** — Mac App Distribution / Installer 인증서, 프로비저닝 프로파일, App Store Connect 앱 레코드
- [ ] **T5.B.2** — 개인정보 처리방침(수집: 앱 사용 집계·일정은 로컬 저장, AI 전송은 사용자 동의 시만), 스토어 설명·스크린샷(ko/en)
  - 참조: linetta `docs/privacy-policy.md`, `docs/store-description-*.md`, `2026-06-18-mac-app-store-submission-design.md`
- [ ] **T5.B.3** — 심사 메모: BYOK 구조, 전체화면 휴식 화면은 사용자가 언제든 건너뛸 수 있음, 로그인 항목은 사용자 동의 후 등록
- [ ] **T5.B.4** — 릴리스 워크플로에 MAS 빌드·`.pkg` 업로드 단계 추가

---

## ✅ Phase 5 Checkpoint

- [ ] 샌드박스 빌드에서 `sandboxd` 위반 로그 없음
- [ ] TestFlight(macOS) 설치 → 휴식·캘린더·AI(API 키) 동작
- [ ] 심사 제출 완료, 반려 시 사유와 대응안을 사용자에게 보고
