# Phase 3: 환경 신호 — 작업지시서

_작성일: 2026-10-20 (rev.2) · 로드맵: [renewal-roadmap.md](renewal-roadmap.md) · 예상 20–30h_

## 페이즈 목표

날씨와 맥 상태를 "보여주기"가 아니라 **행동 제안**으로 연결한다. 끝나면 휴식 화면이 날씨에 맞는 활동을 추천하고, 퇴근 전 비 예보를 알려 주며, 맥이 느릴 때 원인과 정리 후보를 보여 준다. 직접 배포판은 Claude Code 에이전트 상태도 메뉴바에 보인다.

## 전제 조건

- [ ] P2 완료 (MCP 툴 레지스트리 사용)
- [x] P0 스파이크의 "다른 프로세스 조회" 결과 확인 → **FAIL** ([renewal-spike-sandbox.md](renewal-spike-sandbox.md)): `proc_pid_rusage`·`/bin/ps` 모두 샌드박스에서 차단. T3.B.2 프로세스별 목록은 직접판 전용

## 포함 기능

1. 날씨/미세먼지 + 휴식 활동 연동 + 퇴근 비 알림 (양쪽 배포판)
2. 리소스 진단 + 개발 캐시 정리 후보 (앱스토어판은 시스템 전체 지표와 사용자 선택 폴더만)
3. Claude Code 훅 연동 (직접판 전용, 실험, 기본 off)

## 이 페이즈에서 하지 않는 것

- 실시간 그래프 대시보드 (Stats/iStat 영역)
- 파일 삭제·프로세스 종료 자동 실행 (Finder에서 보기 / 확인 후 실행만)

## 작업 체크리스트

### A. 날씨

- [ ] **T3.A.1** — `internal/weather`: Open-Meteo forecast + air-quality(키 불필요). 설정 `weather_location`(도시명 → geocoding 1회). 30분 캐시, 실패 시 마지막 값 + stale. 순수 함수 `Advice(w, now, cfg)` → `walk_ok | indoor | rain_before_leave`.
  - 검증: 녹화한 응답 fixture로 네트워크 없는 테스트
- [ ] **T3.A.2** — 휴식 연동: 휴식 이벤트에 추천 활동 힌트 포함(맑음+미세먼지 보통 이하 → 산책 우선), 퇴근 예정 1시간 전 강수 확률 ≥ 60%면 알림 1회
- [ ] **T3.A.3** — MCP 툴 `get_weather`, 브리핑에 한 줄, 팝오버 "지금" 카드에 아이콘·기온

### B. 리소스 진단

- [ ] **T3.B.1** — `internal/sysinfo`: 시스템 전체 지표만 사용(CPU 사용률 `host_statistics64`, 메모리 압박, 디스크 여유, 배터리) — 양쪽 배포판 공통. 다른 프로세스 정보는 여기서 읽지 않는다. 1분 샘플, 15분 링버퍼. 규칙: 메모리 압박 높음 5분 지속 or 디스크 여유 < 10% → 팝오버 배지(알림은 하루 1회 상한).
  - 검증: 샘플 시퀀스 fixture로 규칙 테스트, `-tags mas` 빌드 확인
- [ ] **T3.B.2** — "왜 느리지?" 패널: 양쪽 공통으로 시스템 지표·스왑과 최근 전면 앱(NSWorkspace, P1 앱 사용 집계)을 결합해 보여 줌. **CPU·메모리 상위 5개 프로세스 목록은 직접판 전용**(`-tags mas`에서는 팩토리가 거부하고 UI 숨김). 양쪽 모두 [활성 상태 보기 열기] 버튼
- [ ] **T3.B.3** — 개발 캐시 정리 후보: 직접판은 `~/Library/Developer/Xcode/DerivedData`, `~/Library/Caches/Homebrew`, Docker 디스크 이미지 등 크기 계산(하루 1회). 앱스토어판은 사용자가 고른 폴더(보안 범위 북마크)만. 각 항목 [Finder에서 보기]만 제공.
- [ ] **T3.B.4** — MCP 툴 `get_system_health`

### C. Claude Code 훅 (직접판 전용)

- [ ] **T3.C.1** — 수신 명령 `<cli> agent-event --kind stop|notification --session <id>` → 엔진 RPC. 활성 에이전트 수·승인 대기 수를 상태에 반영(30분 무소식이면 만료)
- [ ] **T3.C.2** — 설치 도우미 `<cli> integrations claude-code install`: `~/.claude/settings.json` hooks에 항목 추가(기존 내용 보존, diff 출력 후 확인), `uninstall`로 원복
- [ ] **T3.C.3** — 메뉴바 배지(승인 대기 시 점), 휴식 규칙: 휴식 직전 10분 이내이고 에이전트 작업 중이면 "에이전트가 일하는 동안 지금 쉬기" 제안. 설정 `claude_code_integration`(기본 false)

---

## ✅ Phase 3 Checkpoint

**자동:** `make test`, weather/sysinfo fixture 테스트, `go build -tags mas ./...`

**수동:**
- [ ] 맑은 날 휴식 화면에 산책 추천, 미세먼지 나쁨 fixture로 실내 활동 추천
- [ ] 네트워크 끊김 → 날씨 stale 표시, 다른 기능 정상
- [ ] Claude Code 훅 설치 → 권한 요청 시 메뉴바 배지, `uninstall`로 원복

**완료 처리:** 각 기능의 기본값(on/off)을 사용자와 확정.
