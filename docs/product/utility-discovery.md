# 브레이크 유용성 Discovery

- 조사일: 2026-08-19 KST
- 조사 질문: macOS 지식노동자가 알림을 보는 데서 그치지 않고 짧고 의미 있는 휴식을 시작·완료하도록 돕는 가장 작은 제품 흐름은 무엇인가?
- 표기: **[사실]** 저장소/원문으로 확인, **[해석]** 사실에서 도출한 판단, **[가정]** 후속 검증 필요, **[미검증]** 현재 증거 없음.

## 요약 결론

**수정 후 진행**을 권고한다. 우선 가설의 “상황 맞춤 추천 + 메뉴바/대시보드” 전체는 증거와 구현 범위에 비해 크다. 대신 단일 MVP를 **휴식 시작 시 하나의 기본 활동(2분 서서 목·어깨 스트레칭)을 제시하고, 같은 네이티브 휴식 화면 안에서 `시작 → 카운트다운 → 완료`까지 마치는 흐름**으로 제한한다.

마이크로브레이크 메타분석은 활력과 피로에 작지만 유의한 효과를 보고했지만 전체 수행 효과는 유의하지 않았다.[1] 별도의 2025년 메타분석은 컴퓨터 프롬프트가 사무직 노동자의 앉아 있는 시간을 줄일 수 있다고 보면서도 장기·대규모 연구가 더 필요하다고 결론 냈다.[6] 따라서 생산성 향상을 약속하지 않고 “실제 휴식 완료 마찰 감소”를 목표로 삼아야 한다. 경쟁 제품은 사전 예고, snooze/skip, 유휴 감지, 일정 맞춤을 이미 제공한다.[2][3][4] 추천 개수나 개인화보다 **한 화면에서 한 행동을 시작하고 끝내는 결정적 흐름**이 이 제품의 현재 구조와 한 사이클 범위에 더 잘 맞는다.

## 1. 사용자 문제와 행동 장벽

### 확인된 사실

1. **[사실·A]** 2022년 체계적 문헌고찰/메타분석(22개 연구)은 마이크로브레이크가 활력(d=0.36)과 피로(d=0.35)에 작지만 유의한 효과를 보였고, 수행에 대한 전체 효과(d=0.16)는 유의하지 않았다고 보고했다.[1]
2. **[사실·B]** 같은 논문은 22개 연구 중 낮은 비뚤림 위험으로 평가된 연구가 4개뿐이라 전체 위험이 다소 불명확하다고 명시한다.[1]
3. **[사실·B]** 2025년 사무직 컴퓨터 프롬프트 메타분석은 18개 연구·1,164명을 포함했고, 앉아 있는 시간 감소에는 효과가 있다고 결론 냈지만 다양한 업무·건강 결과를 판단하려면 더 큰 장기 연구가 필요하다고 밝혔다.[6]
4. **[사실·A, 저장소]** 현재 앱은 50분 작업/10분 휴식, 5분 전 경고, 자연 휴식 감지, snooze, 강제 휴식, 네 가지 TUI 가이드 활동을 이미 갖고 있다(`README.md:14-25,78-103`, `internal/timer/timer.go:114-208`, `cmd/break-reminder/break.go:12-57`).
5. **[사실·A, 저장소]** 풀스크린 `BreakScreenApp`은 다섯 활동 문구 중 하나를 무작위로 보여 주지만, 이는 실행 가능한 활동이 아니라 텍스트이며 화면의 유일한 동작은 일정 시간 후 skip이다(`helpers/Sources/BreakScreenApp/main.swift:28-34,148-176`).
6. **[사실·A, 저장소]** 네 가지 실제 가이드 활동은 Bubble Tea TUI로만 시작되며(`cmd/break-reminder/break.go:34-50`), 메뉴바와 GUI 대시보드는 `Reset`/`Force Break`만 제공한다(`helpers/Sources/MenuBarApp/main.swift:101-119`, `helpers/Sources/DashboardApp/TimerTabView.swift:78-84`).
7. **[사실·A]** Apple은 알림을 “한눈에 이해할 수 있는 시의적절하고 가치 높은 정보”로 정의한다.[5] 알림 자체가 휴식 완료를 보장한다는 근거는 아니다.

### 해석

- **알림 피로/무시**: 현재 제품은 시작·5분 전·활동 중 경고·종료 알림까지 발생할 수 있다. 반복 알림이 무시된다는 직접 사용자 데이터는 없으므로 현상은 **[가정]**이지만, 알림 수를 늘리는 MVP는 성공 결과와 거리가 멀다.
- **시작 마찰**: 추천 문구와 실행 가능한 가이드가 서로 다른 프로세스/표면에 분리돼 있다. 휴식 시점의 사용자가 CLI 명령을 기억해 터미널에서 실행해야 하므로, 저장소에서 직접 확인되는 가장 강한 마찰이다.
- **완료 마찰**: 현재 가이드 TUI의 시간이 끝나면 `tea.Quit`이 아니라 다음 command만 중단한다(`internal/dashboard/activities.go:32-45,99-108,187-196,278-287`). 자동 종료 여부는 실제 런타임 검증이 더 필요하며, 완료 상태가 명시적이지 않다.
- **선택 부담**: 네 활동을 모두 노출하면 어떤 것이 상황에 맞는지 다시 결정해야 한다. 선택 부담이 실제 이탈 원인이라는 사용자 연구는 **[미검증]**이므로 MVP는 개인화 알고리즘 대신 하나의 안전한 기본값으로 검증한다.
- **방해감**: 경쟁 제품도 사전 예고·snooze·skip·상황별 자동 일시정지를 핵심으로 내세운다.[2][3][4] 강제성이 높을수록 휴식 준수 가능성은 커질 수 있지만 회의·녹화·전체화면 작업을 방해할 위험도 커진다.

## 2. 시장/경쟁 비교

제품 공식 문서/사이트의 현재 기능을 비교했다. 마케팅 효능 주장은 독립 임상 근거로 취급하지 않았다.

| 제품 | 제품 중심 | 확인된 휴식 UX | 회피/상황 대응 | 본 제품에 주는 신호 |
|---|---|---|---|---|
| Stretchly | 크로스플랫폼 오픈소스 휴식 알림 | 기본 10분마다 20초 미니 휴식, 두 번 뒤 5분 긴 휴식; 사전 예고; 휴식 아이디어 표시[2] | 휴식 시작 뒤 제한된 postpone/skip; 5분 유휴 및 DND 때 일시정지[2] | 아이디어만이 아니라 준비 시간과 회피 정책이 함께 있어야 함 |
| BreakTimer | 크로스플랫폼 주기적 휴식 관리 | 간격/길이, 알림 또는 전체화면, 근무시간, 메시지 커스터마이즈[3] | 시작 전 알림에서 빠른 skip/snooze[3] | 추천 기능 자체는 차별점이 약함; 낮은 마찰이 기본 기대치 |
| LookAway | macOS 전용 적응형 휴식 | 사전 예고, 메뉴바 상태/제어, 커서 추종 카운트다운, 타이머·메시지 커스터마이즈[4] | 화면 녹화·회의/통화·영상·집중 앱·전체화면 게임 중 자동 일시정지[4] | “상황 맞춤”은 단순 활동 랜덤화보다 방해 회피에 가까운 강한 경쟁 기준 |
| Break Reminder(현재) | 로컬 Go 타이머 + Swift 네이티브 UI | 전체화면에 무작위 활동 문구, 별도 CLI TUI 가이드 4종 | 유휴 감지, 작업시간, pause 모드, snooze | 이미 가진 활동 콘텐츠를 시작·완료 흐름으로 연결하는 것이 최소 공백 |

**시장 해석**: 사전 예고, skip/snooze, 일정, 유휴 대응은 위 세 경쟁 제품 모두 또는 다수가 제공하는 범용 패턴이다. “상황 맞춤 활동 추천”만으로는 차별성이 입증되지 않았다. 현재 제품의 현실적 차별점은 로컬 우선 구조와 기존 네이티브 break overlay를 이용해 **추천을 즉시 수행 가능한 작은 의식(ritual)**으로 바꾸는 데 있다. 이 차별성의 사용자 선호는 **[미검증]**이다.

## 3. 콘셉트 후보 점수표

5점이 가장 좋다. 구현 크기/위험은 점수가 높을수록 작고 낮다. 총점은 `가치 30% + 차별성 15% + 작은 구현 20% + 낮은 위험 15% + 구조 적합 20%`의 가중 합이다.

| 후보 | 사용자 가치 | 차별성 | 작은 구현 | 낮은 위험 | 구조 적합 | 총점 | 판단 |
|---|---:|---:|---:|---:|---:|---:|---|
| A. 네이티브 휴식 화면의 단일 2분 스트레칭 `시작→완료` | 5 | 3 | 4 | 4 | 5 | **4.35** | 추천 |
| B. 메뉴바/대시보드에 네 활동 빠른 실행 | 4 | 2 | 3 | 3 | 3 | 3.15 | 보류: 표면 2개와 TUI 실행 경계 |
| C. 회의/녹화/집중 앱 감지로 자동 지연 | 4 | 3 | 1 | 2 | 2 | 2.45 | 보류: 권한·앱 감지·정책 범위 큼 |
| D. 완료 기록·스트릭·통계 | 3 | 2 | 2 | 2 | 2 | 2.25 | 기각: 행동 전환보다 측정 기능, 상태 writer 위험 |

### 점수 근거와 반증

- A는 기존 `BreakScreenApp`의 타이머와 활동 문구를 재사용하고 계정/API/새 권한이 없다. 다만 스트레칭의 최적성·2분 길이는 직접 검증되지 않았다. 메타분석은 짧은 휴식의 웰빙 효과를 지지하지만 활동별 최적 처방이나 일반 수행 향상을 확정하지 않는다.[1]
- B는 PM의 초기 가설에 가장 가깝지만 메뉴바에서 현재 Bubble Tea TUI를 실행할 사용 가능한 터미널이 보장되지 않고, Dashboard와 MenuBar 양쪽을 바꾸면 단일 수직 슬라이스가 아니다.
- C는 LookAway가 명확히 경쟁하는 영역이지만[4], 새 시스템 관찰/권한과 다양한 앱 예외 정책이 필요하다.
- D는 성공 측정에는 유용하지만 `State` 필드 확대와 Go/Swift 양쪽 직렬화가 필요해 기존 TIDY-002(다중 writer/필드 유실)를 악화시킬 수 있다.

## 4. 정확히 하나의 추천 MVP

### 이름

**Start One Guided Break — 2분 서서 목·어깨 스트레칭**

### 사용자 흐름

1. 기존 타이머가 휴식으로 전환한다.
2. 기존 네이티브 휴식 화면은 무작위 문구 대신 하나의 명확한 카드 `2분 동안 서서 목과 어깨를 풀어보세요`와 기본 초점 버튼 `시작`을 보여 준다.
3. 사용자가 시작하면 같은 화면에서 2분 카운트다운과 짧은 단계 안내를 본다. 새 앱/터미널은 열지 않는다.
4. 2분이 끝나면 `완료` 결과를 보여 주고 닫는다. 사용자는 Esc/Skip으로 언제든 취소할 수 있다.
5. 원래 10분 휴식 타이머 상태는 Go `timer.Tick`이 계속 소유한다. MVP는 활동 완료를 휴식 전체 완료로 오인하거나 원격 텔레메트리로 보내지 않는다.

### 성공의 대리 관측

- 기능 성공: `휴식 전환 → 활동 카드 표시 → 시작 → 120초 → 완료`가 하나의 Swift 프로세스에서 결정적으로 재현된다.
- 유용성 가설: 한 번의 버튼으로 가이드 휴식을 시작하고 명시적 완료 결과에 도달한다. 실제 장기 준수율 향상은 **[미검증]**이며 이 MVP만으로 주장하지 않는다.
- 접근성: 버튼은 키보드 기본 동작과 의미 있는 접근성 레이블을 제공하고, 애니메이션/색만으로 상태를 전달하지 않는다.

### 30분 구현 카드 단위(한 수직 슬라이스)

1. BreakScreen 인자/순수 모델에 `guided activity` 고정값과 단계/남은 시간 정의.
2. HelperCore에 시작 전/진행/완료 상태 전이와 단위 테스트.
3. `BreakScreenApp`에 단일 카드, Start/Cancel, 진행/완료 UI 연결.
4. VoiceOver 레이블·키보드 기본 초점·Escape/Skip 회귀 수동 검증.
5. Go overlay 인자 전달과 `internal/breakscreen` 테스트; helper가 현재 check를 block하는 동안 동일 launchd job의 60초 firing이 missed되어도 Go 상태를 Swift가 쓰지 않는지 검증.
6. 전체 Go/Swift 회귀와 실제 120초 완료 수동 검증.

### 비범위

- 활동 여러 개 선택, 랜덤/AI/개인화 추천
- 메뉴바와 Dashboard 양쪽에 새 기능 추가
- 회의/녹화/앱 감지, 새 macOS 권한
- 새 설정 키·상태 필드·완료 스트릭·분석/원격 텔레메트리
- 계정, 클라우드, 유료 API
- 기존 SEC-001/SEC-002 또는 TIDY-002를 이 MVP 명목으로 수정
- 릴리스, 버전 범프, 배포

## 5. 기술 검증

### 실제 연결 지점

| 계층 | 현재 연결 | MVP 변경 예상 | 테스트 위치 |
|---|---|---|---|
| 타이머 | `timer.Tick`이 `EnterBreak`와 `ActionNotifyBreakTime` 생성 (`internal/timer/timer.go:135-155`) | 상태 로직 변경 없음 | `internal/timer/timer_test.go` 기존 49.5분→break 회귀 |
| 액션 | `executeActions`가 `breakscreen.Show` 호출 (`cmd/break-reminder/check.go:78-99`) | 호출 시 고정 가이드 메타 전달 여부만 검토 | `cmd/break-reminder/check_test.go` fake 경계 추가 가능 |
| Go→Swift | `showOverlay`가 duration/skip/stats를 CLI 인자로 전달 (`internal/breakscreen/overlay_darwin.go:14-44`) | 고정 activity id/phase를 인자로 추가하거나 helper 기본값 사용 | `internal/breakscreen/breakscreen_test.go`, ArgsParser tests |
| 순수 Swift | `HelperCore`에 Args/Progress/TimeFormatter 존재 | 활동 상태 전이를 순수 타입으로 추가 | `helpers/Tests/HelperCoreTests/GuidedBreakTests.swift` |
| UI | `BreakScreenApp`가 타이머·무작위 텍스트·skip 렌더 (`helpers/Sources/BreakScreenApp/main.swift:114-215`) | 무작위 문구를 단일 시작/진행/완료 UI로 교체 | Swift build + 120초 수동 E2E + VoiceOver |
| 상태 | Go `state.Update/Save`, `EnterBreak`가 소유 (`internal/state/state.go:71-81,425-445`) | 새 상태 필드 없음; Swift writer 추가 금지 | `internal/state/state_test.go` 무변경 회귀 |

### 60초 check/launchd 제약

- 기본 config와 LaunchAgent 모두 60초다(`internal/config/defaults.go:27`, `internal/launchd/launchd.go:246-247`). 휴식 진입은 임계값 도달 후 최대 한 check 주기만큼 늦을 수 있다.
- 가이드의 1초 카운트다운은 Swift 프로세스가 소유한다. Go `check`는 `showOverlay`의 `cmd.Run()`에서 helper 종료까지 block하므로 overlay 중 휴식 상태/통계를 60초마다 동시에 갱신하지 않는다. 가이드 완료 시 Swift가 state 파일을 쓰면 안 된다.
- macOS `StartInterval`은 job 실행 중 firing을 missed한다. 상태는 helper 시작 전에 `break`로 저장되고, helper 종료 뒤 다음 interval에서 Go check가 재개된다. 실제 LaunchAgent 재개 시점과 긴 gap의 통계 정확성은 수동 통합 QA 전까지 **[미검증]**이다.

### 접근성·프라이버시·방해 위험

- **접근성**: 현재 AppKit UI는 수동 frame 배치이며 명시적 accessibility label이 보이지 않는다. Start/Cancel/완료 상태에 VoiceOver 이름·역할·초점 순서를 명시해야 한다.
- **프라이버시**: 추천이 고정 로컬 콘텐츠이고 새 관찰/네트워크가 없으므로 추가 데이터 위험은 낮다. 활동 완료 기록을 상태/로그에 추가하지 않는 것이 MVP 원칙이다.
- **방해**: `.screenSaver` level 다중 모니터 overlay는 강한 방해다(`helpers/Sources/BreakScreenApp/main.swift:72-85`). 기존 notify 모드와 Esc/Skip을 유지하고, 시작 전 카드는 자동으로 신체 활동을 강요하지 않는다.
- **안전**: 의학적 치료나 생산성 향상을 주장하지 않는다. 통증/어지러움이 있으면 중단할 수 있는 문구를 고려한다.

## 6. 반대 증거와 실패 가능성

1. 메타분석의 수행 효과는 유의하지 않았고 비뚤림 위험도 불명확하다.[1] “더 생산적”이라는 제품 약속은 근거 부족이다.
2. 현재 전체화면은 이미 활동 문구와 카운트다운을 제공한다. Start 버튼 추가가 실제 행동을 바꾸지 않고 장식에 그칠 수 있다.
3. 사용자는 휴식 활동보다 회의/녹화 시 방해 회피를 더 중요하게 여길 수 있다. LookAway는 이를 핵심 기능으로 내세운다.[4]
4. 한 가지 스트레칭만 반복하면 금방 무시될 수 있다. 반대로 여러 선택지를 넣으면 결정 마찰이 다시 생긴다. 어느 쪽이 우세한지 사용자 데이터가 없다.
5. `BreakScreenApp`의 기존 무작위 활동과 CLI 가이드가 중복 콘텐츠 소유권을 갖는다. MVP 후 통합하지 않으면 유지보수 중복이 커질 수 있다.
6. block 모드를 쓰지 않는 기존 notify 사용자는 MVP를 보지 못한다. 이번 한 사이클에서 표면을 늘리지 않는 대신 도달 범위를 의도적으로 제한한다.

## 7. 가정과 미검증 사항

- [가정] 주 사용자는 macOS 장시간 지식노동자/개발자다. 저장소 포지셔닝과 맞지만 사용자 인터뷰/사용량 데이터는 없다.
- [수정 확인] 유용성은 알림 횟수가 아니라 실제 짧은 휴식의 시작·완료 마찰 감소로 정의한다.
- [확인] 기존 Go+Swift, 로컬 state/config만으로 단일 흐름을 만들 수 있다.
- [기각] 이 사이클에서 상황 맞춤 개인화, 여러 활동 선택, 메뉴바+대시보드 동시 제공은 범위가 크고 증거가 약하다.
- [미검증] 2분 스트레칭이 이 사용자군의 최적 활동/길이인지.
- [미검증] block-mode 사용자 비율, snooze/skip 빈도, 실제 휴식 완료율.
- [미검증] helper가 120초 block하는 동안 launchd 실행/상태 갱신의 실제 동작.

## 8. 증거 신뢰도와 결정 영향

| 주장 | 등급 | 이유 | 결정 영향 |
|---|---|---|---|
| 짧은 휴식은 활력/피로에 작은 이점, 컴퓨터 프롬프트는 좌식시간 감소 가능 | B | 독립 메타분석 2개가 방향을 지지하지만 장기·광범위 결과는 불확실 | 웰빙/행동 중심 문제는 진행 가능 |
| 수행 향상 | D/반증 | 전체 효과 유의하지 않음 | 생산성 문구 금지 |
| 경쟁 제품의 사전예고/snooze/상황 대응 | A(기능 존재) | 공식 사이트/README 직접 확인 | 알림 추가만으로 차별화하지 않음 |
| 현재 앱의 시작·완료 연결 단절 | A | 실제 코드 경계 확인 | 네이티브 단일 흐름 우선 |
| 단일 기본 활동이 선택 부담을 줄여 완료율 향상 | C | 합리적 추론이나 제품 사용자 데이터 없음 | 작은 MVP로만 검증 |

## 권고

**수정 후 진행**: 정확히 하나의 MVP인 “Start One Guided Break”만 Definition 단계로 넘긴다. PM/Planner는 생산성 향상이나 개인화 효과를 확정 사실로 쓰지 말고, `한 화면·한 기본 활동·한 번의 시작·명시적 완료·언제든 취소`를 수용 기준으로 고정해야 한다.

### 출처 메타데이터

| ID | 제목/종류 | 게시·업데이트일 | 접근일 |
|---|---|---|---|
| [1] | “Give me a break!” 체계적 문헌고찰/메타분석 | 2022-08-31 | 2026-08-19 KST |
| [2] | Stretchly 공식 저장소 README | 문서에 개별 게시일 없음 | 2026-08-19 KST |
| [3] | BreakTimer 공식 사이트 | 페이지에 게시일 없음 | 2026-08-19 KST |
| [4] | LookAway 공식 사이트 | 페이지에 게시일 없음 | 2026-08-19 KST |
| [5] | Apple HIG Notifications | 페이지에 게시일 없음 | 2026-08-19 KST |
| [6] | 컴퓨터 프롬프트와 좌식 휴식 체계적 문헌고찰/메타분석 | 2025-06-13 | 2026-08-19 KST |

## Sources

[1] https://pmc.ncbi.nlm.nih.gov/articles/PMC9432722 — Give me a break! systematic review and meta-analysis
    > "Overall, only four out of twenty-two studies were assessed with a low risk of bias, whereas one presented a high risk for at least half of the criteria. Thus, we are inclined to consider the risk of bias in our overall sample as being somewhat unclear."
    > "These revealed a statistically significant but small effect of micro-breaks on vigor, d = 0.36, p < .001, 95% CI [.16, .55], and fatigue d = 0.35, p < .001, 95% CI [.19, .50], while the effect on performance was not statistically significant, d = 0.16, p = .17, 95% CI [-0.04, .37]."
[2] https://github.com/hovancik/stretchly/blob/trunk/README.md — Stretchly README
    > "By default, there is a 20 second Mini break every 10 minutes and a 5 minute Long break every 30 minutes (after 2 Mini breaks)."
    > "When a break starts, you can postpone it once for 2 minutes (Mini breaks) or 5 minutes (Long breaks). Then, after a specific time interval passes, you can skip the break."
    > "Stretchly is monitoring your idle time, so when you are idle for 5 minutes, breaks will be paused until you return."
[3] https://breaktimer.app — BreakTimer official site
    > "BreakTimer lets you set how often breaks happen and how long each one lasts, so breaks fit the way you work."
    > "BreakTimer lets you know when breaks are about to start, so you can quickly skip or snooze if timing is tight."
[4] https://lookaway.com — LookAway official site
    > "Automatically pauses during"
    > "LookAway waits for the right moment to show a break reminder, and gives you a heads-up beforehand."
[5] https://developer.apple.com/design/human-interface-guidelines/notifications — Apple HIG Notifications
    > "A notification gives people timely, high-value information they can understand at a glance."
[6] https://pmc.ncbi.nlm.nih.gov/articles/PMC12164069 — Computer prompts and sedentary breaks: systematic review and meta-analysis
    > "Computer prompt software interventions show effectiveness in reducing sitting time among office workers. However, more long-term prospective studies with larger sample sizes are needed to accurately determine the effectiveness of computer prompts on various work- and health-related outcomes."
    > "From 17,880 records, 18 studies involving 1164 office workers were included in the analysis."
