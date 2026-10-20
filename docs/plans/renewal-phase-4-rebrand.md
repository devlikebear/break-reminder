# Phase 4: 리브랜딩·직접 배포 — 작업지시서

_작성일: 2026-10-20 (rev.2) · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 20–30h_

## 페이즈 목표

새 이름의 공증된 `.app`을 Homebrew cask로 배포하고, v0.16 사용자의 설정·기록을 자동으로 옮기며, launchd 60초 `check` 구조를 퇴역시킨다. 끝나면 `brew install --cask` 한 번으로 새 앱이 동작하고 `break-reminder` 명령은 호환 별칭으로 남는다.

## 전제 조건

- [ ] 새 이름 확정 (GitHub·Homebrew·번들 ID·App Store 이름 중복 확인)
- [ ] P0~P2 완료 (P3는 선택)

## 작업 체크리스트

### A. 이름 적용

- [ ] **T4.A.1** — `internal/brand`(앱 이름, 번들 ID 접두사, 데이터 디렉터리명, MCP 서버 이름). `internal/paths`가 이를 사용
- [ ] **T4.A.2** — i18n 카탈로그·README·CHANGELOG·아이콘(햄스터 유지) 교체

### B. 구조 정리

- [ ] **T4.B.1** — 휴식 화면을 별도 `break-screen` 프로세스에서 앱 프로세스 안의 창으로 통합(샌드박스에서 헬퍼 실행 줄이기), Dashboard도 앱 안 창으로
- [ ] **T4.B.2** — launchd `check`·time-tools 워커·`menubar` LaunchAgent 퇴역. CLI는 엔진 HTTP RPC 클라이언트로 전환(엔진이 꺼져 있으면 앱 실행 안내)
- [ ] **T4.B.3** — `terminal-notifier`, `ioreg`, `say` 의존 제거 확인 (`doctor` 항목 정리)

### C. 마이그레이션

- [ ] **T4.C.1** — 첫 실행 시 `~/.config/break-reminder/*`, `~/.break-reminder-state`를 새 경로로 복사(원본 보존), 기존 LaunchAgent 언로드·제거. 기존 `service migrate-runtime`의 "중지한 기능은 중지 상태 유지" 규칙 계승
- [ ] **T4.C.2** — `break-reminder` 호환 심볼릭 링크와 사용 중단 안내 (2개 릴리스 유지)
- [ ] **T4.C.3** — MCP 서버 이름 변경 시 Claude Code 재등록 안내

### D. 배포

- [ ] **T4.D.1** — `release.yml`: `make app` → Developer ID 서명 → `notarytool` 공증 → staple → zip/dmg 첨부
  - 참조: linetta 직접 배포 워크플로
- [ ] **T4.D.2** — `devlikebear/tap`에 cask 추가, 기존 `break-reminder` formula는 cask 안내로 교체
- [ ] **T4.D.3** — 업데이트: cask 기반 기존 자동 업데이트(`internal/autoupdate`)를 새 경로로 교체 (Sparkle 도입은 사용자와 상의)

---

## ✅ Phase 4 Checkpoint

**자동:** `make test`, `make app`, 가짜 HOME에 v0.16 데이터 fixture로 마이그레이션 테스트

**수동:**
- [ ] v0.16.x가 설치된 맥에 cask 설치 → 설정·통계·타이머 유지, 이전 LaunchAgent 제거됨
- [ ] 기존에 중지한 기능은 중지 상태 유지
- [ ] `spctl -a -vv`에서 notarized Developer ID 확인
- [ ] `break-reminder status` 호환 별칭 동작

**완료 처리:** 릴리스 노트 초안을 사용자에게 보여 주고 승인 후 Release 워크플로 실행.
