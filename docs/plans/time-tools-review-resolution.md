# 시간 도구 개발계획 — 리뷰 반영 기록

작성일: 2026-09-28 · 대조 기준 HEAD: `8e8cb79` · 상태: 리뷰 반영 완료. 후속 Phase 1/2 구현·검증은 [검증 기록](time-tools-validation.md) 참조.

## 검토 원칙

리뷰의 코드 전제를 현재 소스와 대조했다. 기존에 정의된 항목은 로드맵/단계별 지시서에서도 바로 찾을 수 있게 보강했고, 업그레이드 최초 전환·알림 fallback·전체화면 UX는 새 계약으로 추가했다.

## 중요 지적별 처리

| 리뷰 | 판정과 반영 | 문서/검증 |
|---|---|---|
| 1. 첨부 문서 없음 | 현재 작업 트리에는 기존 8개 모두 존재, Git에는 미추적. 리뷰가 본 commit에서 없는 것은 맞다. 리뷰 기록 포함 9개 전달 manifest와 같은 커밋 포함 조건을 추가. **Git 전달 완료는 아직 아님** | [로드맵](time-tools-roadmap.md) 6절 |
| 2. worker 생명주기 | 별도 agent는 기존 설계에 있었으나 Aqua/KeepAlive/stop/logout 계약과 첫 업그레이드 경로가 부족했다. 명시 보완. 구버전 updater 첫 전환은 1회 설정, 새 updater의 이후 전환은 새 바이너리 migration | [설계](time-tools-design.md) 6절, P2.5, U01–U05 |
| 3. sleep 뒤 지연 | 기존 wall-clock 계약을 코드 수준으로 명확화: 약 1초마다 새 UnixMilli 비교, startup 즉시 reconcile, 장시간 Timer/Sleep 완료 처리 금지 | 설계 5절, P2.4, T15와 실제 wake QA |
| 4. notifier 대비책 | 기존 완료 event/card를 fallback으로 확정. notifier 실패 시 소리도 보장하지 않음. 기존 doctor fail에 복구 안내·worker 상태를 보강. osascript는 추가하지 않음 | 설계 6절, P2.6, T13 |
| 5. 별도 state | 기존 별도 JSON+flock 계약을 로드맵 결정 5에 승격. 기존 Reset/Force Break와의 독립성 회귀 테스트 추가 | 로드맵 2절, 설계 4절, T16 |
| 6. 동시 조작 | 기존 ID/revision 계약 유지. A 취소→B 시작→옛 A pause 재현 시나리오 추가. revision 생략해도 target ID 검증 | P2.6, T06 |
| 7. 포모도로 시작 | 기존 on→start 순차 계약을 Phase 1 도입부에서 정의. 모드 선택만 하는 동작과 구분, 부분 성공 재시도는 start만 | [Phase 1](time-tools-phase-1-pomodoro.md), P01/P03 |

## 작은 지적별 처리

- 첫 배포 1주 사용 후 Phase 3/4 각각 go/no-go 도입. 스톱워치의 통계 합산 없는 기록 가치와 알람 고유 수요를 확인한다.
- Phase 3의 의존은 store/CLI/UI 공유이며 worker 불필요. Phase 4는 Phase 3 없이도 가능하다.
- `.analysis/` tracked/untracked 혼재로 문구 수정.
- 배너가 전체화면 뒤에 가려지는 문제는 QA만 추가하지 않고 primary 휴식 화면의 읽기 전용 완료 안내로 제품 계약을 보완. 휴식·가이드 진행과 확인 상태는 바꾸지 않는다.
- Phase 2는 28–44→32–50시간, 첫 배포는 40–64→44–70시간으로 조정. P2.5 설치/전환과 P2.6 overlay/fallback 공수에 반영했다.

## 계획 리뷰 당시 범위와 후속 작업

계획 리뷰 시점의 결과는 문서 9개의 로컬 반영이었다. 소스·서비스·사용자 상태를 수정하지 않았고 commit/push도 수행하지 않았다. 문서 파일/상대 링크/형식은 점검하며 제품 실행 테스트는 NOT RUN이다. 다른 체크아웃/원격에서 구현을 시작하려면 문서 묶음 커밋 전달이 필요하다.

기존 작업 규칙과 새 manifest를 따르면 누락 여부를 검증할 수 있다. Git 공유 완료 전에 원격 handoff 완료라고 보고하지 않는다.

2026-09-29 후속: 사용자 릴리즈 요청에 따라 Phase 1/2 소스와 테스트를 구현했으며, 이 문서 묶음을 구현과 함께 Git에 포함한다. Phase 3/4 go/no-go는 유지한다.
