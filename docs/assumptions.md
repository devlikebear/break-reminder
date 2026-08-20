# 제품 가정 및 의사결정 로그

이 문서는 되돌릴 수 있는 제품 가정의 상태를 기록한다. 조사 근거의 상세 내용과 출처는 `docs/product/utility-discovery.md`를 따른다.

## 2026-08-19 — 유용성 Discovery

| ID | 초기 가정 | 상태 | 조사 후 결정 | 근거/불확실성 |
|---|---|---|---|---|
| U-001 | 주 사용자는 macOS에서 장시간 일하는 지식노동자/개발자다. | **유지(미검증)** | 첫 MVP의 대상과 macOS 14+ 범위는 유지한다. | 제품 포지셔닝·코드 구조와 일치하지만 인터뷰, 설치 분석, 사용자 구성 데이터가 없다. |
| U-002 | 유용성의 1차 결과는 알림 횟수가 아니라 실제 짧고 의미 있는 휴식의 시작·완료 용이성이다. | **확인** | 기능 성공을 `휴식 전환 → 시작 → 카운트다운 → 완료`의 관측 가능한 흐름으로 정의한다. 장기 행동 효과는 별도 검증한다. | 마이크로브레이크는 활력/피로에 작은 이점이 있으나 수행 향상은 유의하지 않았다. 현재 코드는 알림/문구와 실제 가이드 실행이 분리돼 있다. |
| U-003 | 한 사이클 MVP는 기존 Go+Swift와 로컬 상태/설정만 사용하는 단일 end-to-end 흐름이다. | **확인** | BreakScreenApp 내부 단일 가이드 흐름으로 제한한다. 계정/API/새 권한/원격 텔레메트리는 추가하지 않는다. | 현재 timer→breakscreen→Swift helper 경계를 재사용할 수 있다. 새 상태 필드 없이 구현 가능하다. |
| U-004 | 휴식 시점에 상황에 맞는 짧은 활동을 추천하고 메뉴바/대시보드에서 한 번에 시작한다. | **수정** | “상황 맞춤”과 복수 표면은 보류한다. 기존 break overlay에서 하나의 고정 2분 스트레칭을 한 번에 시작·완료하는 MVP로 축소한다. | 개인화·선택 UI의 우월성은 미검증이다. 메뉴바에서 기존 TUI 활동 실행도 터미널 경계 때문에 안전한 재사용이 아니다. |
| U-005 | 릴리스/배포/버전 작업은 이번 사이클 밖이다. | **확인** | 문서·Definition·설계·구현 검증까지만 진행하고 릴리스는 별도 카드로 둔다. | PM 범위 결정. |
| U-006 | 활동 완료/스트릭을 상태에 기록하면 유용성 검증에 필요하다. | **기각** | MVP는 결정적 UI 완료 결과로 검증하며 새 state 필드를 추가하지 않는다. | Go/Swift 다중 writer와 필드 유실 위험(TIDY-002)을 키우고, 원격 분석도 범위 밖이다. |
| U-007 | 활동 추천만 추가하면 차별화된다. | **기각** | 차별화 주장은 하지 않는다. 경쟁 제품도 break ideas, 커스텀 메시지, 적응형 timing을 제공한다. | 공식 제품 기능 비교. 현재의 기회는 콘텐츠 수보다 시작·완료 연결의 작은 마찰 제거다. |
| U-008 | 2분 서서 목·어깨 스트레칭이 최적 기본값이다. | **신규 가정** | 안전하고 짧은 기본값으로 한 사이클만 검증한다. | 마이크로브레이크 일반 효과는 있으나 이 활동/길이가 본 사용자군에 최적이라는 직접 증거는 없다. |
| U-009 | block-mode 네이티브 화면이 첫 검증 표면으로 충분하다. | **신규 가정** | 도달 범위를 의도적으로 block mode로 제한하고 메뉴바/notify 표면 확장은 후속 판단한다. | 단일 수직 슬라이스를 지키기 위한 범위 선택. 실제 block-mode 사용 비율은 미검증이다. |

## 다음 단계에 전달할 결정

1. 추천 MVP는 정확히 하나: `Start One Guided Break — 2분 서서 목·어깨 스트레칭`.
2. 한 화면에서 하나의 기본 활동을 `시작 → 진행 → 완료`하며 Cancel/Escape/기존 Skip을 유지한다.
3. 기존 Go timer/state가 휴식 전체의 소유자다. Swift는 이 MVP를 위해 state 파일을 쓰거나 새 필드를 만들지 않는다.
4. “생산성 향상”, “개인 맞춤”, “완료율 향상”은 검증 전 마케팅/PRD 확정 사실로 쓰지 않는다.
5. 120초 helper가 Go check를 block하고 실행 중 60초 launchd firing이 missed되는 계약, state 선저장, helper 종료 뒤 다음 interval 재개를 수용 기준에 넣는다.
6. VoiceOver 의미 있는 레이블, 키보드 기본 초점, 색 이외의 상태 표현을 수용 기준에 넣는다.

## 2026-08-19 — Planner Definition 결정

| ID | 결정 대상 | 상태 | 확정/기각 내용 | 되돌림 조건 |
|---|---|---|---|---|
| P-001 | MVP의 구현 표면 | **확정** | `break_screen_mode=block`의 기존 `BreakScreenApp` primary window만 바꾼다. notify, 메뉴바, Dashboard, TUI는 무변경이다. | block-mode 도달률이 검증에 부족하다는 사용자 근거가 생기면 별도 사이클에서 표면을 재선정한다. |
| P-002 | 활동 콘텐츠와 시간 | **확정(가정 U-008 유지)** | ID `standing-neck-shoulder-stretch-v1`, 활동 120초, 완료 표시 3초를 고정한다. 시작에는 남은 전체 휴식 123초 이상이 필요하다. | 사용자 관찰에서 안전성·이해도·완료 마찰 문제가 확인되면 콘텐츠/시간을 별도 실험으로 바꾼다. |
| P-003 | 상태 스키마 | **확정** | 휘발성 Swift 모델 `GuidedBreakSession`의 `ready/running/completed`만 사용한다. Go `State`, state 파일 key, config, history, log에는 완료/phase를 추가하지 않는다. | 완료의 장기 효과를 측정하기 위한 별도 개인정보·데이터 정책이 승인될 때만 재검토한다. |
| P-004 | Go↔Swift 경계 | **확정** | 기존 `--duration`, `--skip-after`, `--work-min`, `--break-min`을 그대로 사용한다. 새 CLI flag/command/config key는 만들지 않는다. | 활동이 복수화되거나 Go가 콘텐츠를 선택해야 하는 승인된 후속 범위가 생길 때 버전 있는 계약을 새로 정의한다. |
| P-005 | 취소 의미 | **확정** | running의 `가이드 취소`는 overlay를 닫지 않고 ready로 돌아간다. Esc와 활성화된 기존 Skip은 overlay 전체를 닫는다. 무시는 자동 시작 없이 ready를 유지한다. | 사용성 테스트에서 두 종류의 취소가 혼란을 만든다는 근거가 생기면 카피/동작을 재검토한다. |
| P-006 | 짧은 휴식 처리 | **확정** | 남은 전체 휴식이 123초 미만이면 Start를 비활성화한다. 가이드가 전체 휴식을 연장하거나 Go break 상태를 변경하지 않는다. | 향후 짧은 활동 variant가 승인되면 별도 activity ID와 명시적 선택 정책으로 확장한다. |
| P-007 | guided phase 복구 | **확정** | helper 재실행 시 guided phase를 복원·추정하지 않고 항상 ready로 시작한다. 기존 Go `BreakStart`로 계산한 전체 휴식 remaining만 신뢰한다. | 지속 phase가 실제 사용자 가치를 만든다는 근거와 안전한 단일 writer 설계가 함께 승인될 때만 바꾼다. |
| P-008 | 원격/로컬 완료 측정 | **기각** | 이번 MVP에서 완료 이벤트 저장, 스트릭, 분석, 원격 텔레메트리를 만들지 않는다. 결정적 모델 상태와 UI 결과·테스트로 기능 성공만 판정한다. | 사용자 동의·보존 기간·목적이 정의된 별도 측정 계획이 승인될 때 재검토한다. |
| P-009 | 완료 후 동작 | **확정** | 120초 완료 후 3초 동안 텍스트와 VoiceOver announcement를 제공한 뒤 overlay를 닫는다. underlying Go state는 break로 유지한다. | 수동 접근성 QA에서 3초가 낭독에 불충분하면 자동 종료 지연을 조정하되 활동 120초와 state 무변경은 유지한다. |
| P-010 | `break_activities_enabled` 재사용 | **기각** | 기존 설정의 의미를 이 MVP용 토글로 재정의하지 않는다. 새 설정도 추가하지 않는다. | 설정 의미와 기존 사용자 기대를 조사한 후 별도 호환 정책이 승인될 때만 연결한다. |

## 보류 중인 검증 질문

- 사용자가 가장 자주 무시하거나 snooze하는 순간은 언제인가?
- block/notify 모드별 실제 사용 비율은 얼마인가?
- 하나의 기본 활동과 여러 선택지 중 어느 쪽이 시작·완료율이 높은가?
- 2분 스트레칭이 20초 눈 휴식이나 4분 호흡보다 더 적합한가?
- 진행 중인 block helper 때문에 누락된 launchd interval 뒤 상태 통계가 실제 사용 시간 전체를 정확히 보존하는가?

## 2026-08-19 — UX/UI 설계의 되돌릴 수 있는 결정

| ID | 설계 결정 | 현재 선택 | 되돌림 조건 |
|---|---|---|---|
| D-001 | 가이드 정보 구조 | 기존 전체 휴식 타이머를 상위에 유지하고, 기존 무작위 활동 문구 자리를 단일 phase card로 교체한다. | 사용성 관찰에서 전체 휴식 시간과 가이드 시간이 반복적으로 혼동되면 두 시간의 배치·레이블을 재검토한다. |
| D-002 | 완료 표시 밀도 | 완료 문장과 자동 종료 안내만 3초 동안 표시하며 별도 숫자 countdown·애니메이션은 두지 않는다. | VoiceOver 수동 QA에서 3초 내 announcement 이해가 어렵거나 자동 종료 예측성이 낮으면 Planner P-009 조건에 따라 표시 시간을 재검토한다. |
| D-003 | BreakScreen 색 적용 | 새 공용 디자인 시스템을 만들지 않고 현행 dark overlay와 `ThemeManager.accentBreak`의 의미값을 AppKit 색으로 사용한다. | BreakScreen과 Dashboard의 공용 token화가 별도 범위로 승인되면 중복 색 값을 HelperCore가 아닌 적절한 UI 공용 계층으로 통합한다. |
| D-004 | 한국어 가이드 카피 | 새 guided card와 접근성 카피는 PRD의 한국어 계약을 사용하고, 기존 `Skip Break`·secondary `Break Time`은 회귀 방지를 위해 유지한다. | 제품 전체 localization 또는 기존 overlay 카피 통일 작업이 승인되면 한 번에 재검토한다. |

## 2026-08-19 — 구현 후 팩트체크

| ID | 검증 대상 | 상태 | 확인·기각 결과 | 근거/잔여 불확실성 |
|---|---|---|---|---|
| V-001 | 한 화면의 고정 가이드가 시작·완료율을 높인다. | **유지(미검증)** | 모델과 UI adapter는 조작 경로를 연결하지만 실제 행동효과는 확인하지 못했다. | 마이크로브레이크·컴퓨터 프롬프트 메타분석은 일반 방향만 지지한다. 실제 사용자 start/cancel/completion 관찰이 없다. |
| V-002 | overlay 실행 중에도 동일 launchd job의 다음 60초 check가 실행된다. | **기각** | `check`는 `cmd.Run()`으로 helper 종료까지 block하고, macOS `launchd.plist(5)`는 job 실행 중 `StartInterval` firing이 missed 된다고 명시한다. | state는 helper 전에 break로 저장된다. helper 종료 뒤 통계 보존의 실제 장시간 정확성은 별도 검증이 필요하다. |
| V-003 | 구현이 새 원격 전송·완료 기록·state/config write를 추가하지 않는다. | **확인** | 변경 Swift/Go diff에 네트워크, 계정, 식별자, 로그 이벤트, state/config/history writer 또는 새 schema/flag가 없다. | 기존 비범위 Swift writer 문제(TIDY-002)는 그대로이며 이번 변경이 악화하지 않았다. |
| V-004 | P0 keyboard·VoiceOver·123초 실제 runtime이 충족된다. | **유지(미검증)** | AppKit API는 build되고 순수 모델 123 virtual ticks는 통과했지만 실제 UI·낭독·focus·timer drift는 실행하지 않았다. | QA-01~05의 수동 관찰이 필요하다. |

## 2026-08-19 — 최종 품질 게이트

| ID | 최종 확정/잔여 미검증 | 상태 | 결정 | 근거/다음 조건 |
|---|---|---|---|---|
| F-001 | 자동 회귀·빌드·privacy/scope | **확인** | Go/Swift 전체 테스트, race, vet, release build, diff 검사는 통과했고 새 state/config/CLI/network/dependency는 없다. | PM이 현재 working tree에서 명령을 독립 재실행했다. 상세 결과는 `utility-mvp-final-review.md`를 따른다. |
| F-002 | P0 실제 AppKit·키보드·VoiceOver·123초 runtime | **미검증 / 완료 차단** | 소스와 build만으로 P0를 통과시키지 않는다. QA-01~06의 실제 관찰 전 최종 제품 판정은 FAIL이다. | primary/secondary UI, Return/Space/Tab, VoiceOver 1회 announcement, Reduce Motion/Contrast, stopwatch 123초 결과가 필요하다. |
| F-003 | overlay 중 동일 launchd job의 60초 동시 tick | **기각 확정** | `check`가 helper 종료까지 block하므로 실행 중 `StartInterval` firing은 missed 된다. 상태는 먼저 break로 저장되고 다음 Go 갱신은 helper 종료 뒤 재개된다. | PRD/AC/구현 문서의 동시 tick 표현을 이 실제 스케줄링 계약으로 정정해야 한다. |
| F-004 | 유용성·활동 최적성 | **잔여 미검증** | 한 화면 흐름이 실제 start/completion율을 높인다는 인과효과와 2분 목·어깨 활동의 최적성·안전성은 확정하지 않는다. | 별도 사용자 관찰/인터뷰가 필요하며 이번 P0 기능 검증을 대체하지 않는다. |

## 2026-08-20 — 실런타임 QA와 릴리스 결정

| ID | 결정 대상 | 상태 | 결정 | 근거/잔여 |
|---|---|---|---|---|
| R-001 | Guided Break 핵심 실제 흐름 | **확인** | v0.12.0 릴리스 범위로 승인한다. | exact PID/window CuaDriver로 Start/키보드/guard/Cancel/Esc/Skip/`01:00`/completed/자동 종료를 실제 관찰했다. |
| R-002 | 환경 의존 접근성 변형 | **N/A 유지** | 실제 VoiceOver 음성·announcement event count, focus 속성 API, Reduce Motion/Increase Contrast, multi-display를 제품 결함 없이 별도 수동 범위로 둔다. | 현재 QA 환경이 해당 관찰 표면을 제공하지 않는다. |
| R-003 | 릴리스 버전 | **확정** | 신규 사용자 기능이므로 semver minor인 v0.12.0으로 출시한다. | 저장소 versioning 규칙에서 feat 포함 시 minor bump. |
