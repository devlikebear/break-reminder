# Start One Guided Break — 구현 후 팩트체크

- 검토일: 2026-08-19 KST
- 범위: 구현 diff, P0 AC, 외부 근거, 60초 launchd 조건, 접근성·프라이버시·보호 자산
- 판정 표기: `확인`(직접 원문·코드·실행), `부분 확인`, `반증`, `미검증`
- 최종 판정: **FAIL — 자동 검증은 통과했으나 P0 수동 UX·접근성·123초 실제 런타임 증거가 없고, 60초 launchd 동시 tick 가정은 실제 스케줄링과 다르다.**

> **2026-08-20 addendum:** 최초 팩트체크 뒤 exact PID/window CuaDriver QA가 완료됐다. Start/Return/Space, 122/123 guard, Cancel/restart, phase별 Esc, timed Skip, exact `01:00`, 120초 completed와 약 3초 자동 종료는 실제 helper에서 확인됐다. 실제 VoiceOver 음성·announcement event count와 multi-display는 여전히 환경 의존 N/A다. launchd 동시 tick 반증과 blocking/missed-firing 해석은 그대로 유효하다.

## 1. 조사 질문

1. 구현된 고정 2분 가이드가 원래 문제인 “휴식 시작·완료 조작 마찰 감소”에 맞는가?
2. PRD의 사실·해석·가정이 구현 후에도 구분되는가?
3. 코드가 P0 AC, 기존 break activities·snooze·pause·Skip·Esc, macOS 14 조건과 충돌하지 않는가?
4. 과도한 강제, 다크 패턴, 원격 전송, 불필요한 상태 수집을 추가했는가?
5. 최소 2개의 실패·반례가 자동 테스트 또는 실제 런타임으로 확인되는가?

## 2. 요약 결론

구현은 승인된 좁은 범위를 대체로 정확히 따른다. `GuidedBreakSession`은 `ready → running(120) → completed(3) → dismiss`를 휘발성 메모리에서 결정적으로 구현하고, 122/123초 guard·취소·재시작·단계 문구 경계를 10개 단위 테스트로 검증한다. `BreakScreenApp` diff에는 새 네트워크, 파일 write, config/state schema, CLI flag, dependency가 없으며 Esc·기존 Skip·secondary display 분기는 유지됐다.

그러나 이 MVP가 실제 행동을 개선한다는 인과 주장은 여전히 **미검증**이다. 마이크로브레이크 문헌은 웰빙 방향의 가능성을 지지하지만 수행 향상을 지지하지 않고 연구 비뚤림도 불명확하다.[1] 컴퓨터 프롬프트 문헌도 좌식시간 감소 가능성을 보고하지만 장기·광범위 업무·건강 결과에는 추가 연구가 필요하다고 결론 낸다.[2] 어느 문헌도 “고정 2분 목·어깨 카드가 이 제품 사용자의 시작·완료율을 높인다”를 직접 검증하지 않는다.

또한 로컬 macOS `launchd.plist(5)` 원문은 `StartInterval` 시점에 job이 실행 중이면 그 interval이 누락된다고 명시한다. 현재 `check → breakscreen.Show → cmd.Run()`은 helper 종료까지 block하므로, overlay가 열린 동안 동일 launchd job의 60초 Go tick이 동시에 실행된다는 전제는 사실이 아니다. 상태는 overlay 시작 전에 `break`로 저장되지만, 다음 Go 갱신은 helper 종료 뒤 다음 interval까지 지연된다.

## 3. Claim / Evidence / Verdict

| Claim | Evidence | Verdict | 신뢰도 |
|---|---|---|---|
| 마이크로브레이크는 활력·피로에 작은 이점이 있고 일반 수행 향상은 확정되지 않았다. | 22개 독립 표본 메타분석은 웰빙 방향을 지지하되 overall performance 효과는 유의하지 않았고, 22개 중 낮은 비뚤림 위험 연구는 4개뿐이라고 보고한다.[1] | **확인**. 생산성 향상 카피 금지는 타당하다. | B |
| 컴퓨터 프롬프트는 사무직 좌식시간을 줄일 수 있다. | 18개 연구·1,164명 메타분석은 감소 효과를 보고하면서 장기·대규모 연구가 더 필요하다고 명시한다.[2] | **부분 확인**. 프롬프트 일반 효과이지 이 UI·활동·완료율의 직접 근거는 아니다. | B |
| 경쟁 제품은 사전 예고, snooze/skip, 상황 회피를 이미 제공한다. | Stretchly는 기본 주기, postpone/skip, idle/DND pause를 문서화한다.[3] BreakTimer는 사전 알림의 skip/snooze를 제공한다.[4] LookAway는 heads-up과 화면 녹화·회의/통화 중 자동 pause를 제시한다.[5] | **확인(기능 존재)**. 제품사의 효능 마케팅은 독립 효과 근거로 취급하지 않았다. | A(기능) |
| 알림 자체보다 한 화면의 실행 흐름이 실제 휴식 완료율을 높인다. | Apple은 notification을 한눈에 이해할 수 있는 시의적절하고 가치 높은 정보로 정의하지만 완료 행동을 보장한다고 말하지 않는다.[6] 저장소 diff는 표면 간 이동을 제거했다. | **미검증**. 조작 단계 감소는 확인되지만 행동 개선의 인과효과는 사용자 관찰이 필요하다. | C |
| 2분 목·어깨 스트레칭이 최적 기본값이다. | 활동별·길이별 직접 비교 근거가 없고 PRD도 U-008 가정으로 표시한다. | **미검증**. 최적성·안전성·반복 피로를 주장하면 안 된다. | D |
| 구현은 P0 상태 모델과 exact timing을 따른다. | `GuidedBreakSession.swift:14-50`, `GuidedBreakSessionTests.swift:21-123`; 2026-08-19 재실행에서 10개 관련 테스트와 Swift 전체 101개 테스트가 모두 통과했다. | **자동 범위 확인**. 실제 AppKit timer scheduling은 미검증이다. | A(모델) / D(실 UX) |
| 시작·취소·Esc·Skip이 예측 가능하다. | `main.swift:61-67,260-307,342-349,450-483`; cancel/restart unit은 통과했다. | **부분 확인**. 소스·순수 모델은 일치하지만 실제 키보드/포인터 동작은 NOT RUN이다. | B |
| VoiceOver·focus·Reduce Motion P0가 충족된다. | AppKit API는 빌드되며 label/value/help와 announcement 코드가 있다(`main.swift:385-499`). 실제 낭독·focus ring·Tab/Shift-Tab·1회 announcement·3초 이해 가능성은 관찰하지 않았다. | **미검증 / P0 증거 부족**. | D |
| 60초 launchd check가 overlay 중에도 동시에 진행된다. | 기본 config와 plist는 60초이나(`defaults.go:27`, `launchd.go:246-247`), `showOverlay`는 `cmd.Run()`으로 block한다(`overlay_darwin.go:14-44`). 로컬 macOS `launchd.plist(5)`는 실행 중 interval firing이 missed 된다고 명시한다. | **반증**. AC-07의 독립 60-tick 단위 테스트는 논리 계약이지 실제 동시 runtime 증거가 아니다. | A |
| Swift 구현은 새 상태·텔레메트리·원격 전송을 추가하지 않는다. | 변경 Swift 파일과 신규 모델에서 `URLSession`, `FileManager`, `UserDefaults`, write API가 없고 Go state/config/CLI는 diff에 없다. | **확인**. 기존 저장소의 별도 Swift writer 문제는 비범위이며 이번 diff가 악화하지 않았다. | A |
| README/CHANGELOG의 동작 설명이 실제로 확인됐다. | 123초 guard, 상태 전이, Esc/Skip/secondary branch는 코드와 일치한다. “Return/Space 동작”, “한 번 announcement”, “3초 뒤 종료”는 빌드·모델로만 확인되고 실제 UI에서는 NOT RUN이다. | **부분 확인**. 사용자 문서가 수동 검증보다 앞서 확정형으로 작성됐다. | B |

## 4. PRD의 사실·해석·가정 분리 검토

- **사실로 유지 가능**: 기존 추천 문구와 TUI 가이드의 표면 분리, Go state 선저장, 기존 CLI 계약, macOS 14 target, 60초 config/plist, 새 상태·네트워크 없음.
- **해석으로 유지해야 함**: 한 화면·한 행동이 조작 마찰을 줄인다는 판단. 클릭/표면 수 감소는 구조상 사실이지만 실제 시작률·완료율 개선은 아니다.
- **가정으로 유지해야 함**: U-001 대상 사용자, U-008 활동·길이 최적성, U-009 block-mode 도달 범위, 반복 사용 시 알림 피로·선택 부담 감소.
- **기각/수정 필요**: overlay 실행 중에도 다음 60초 launchd check가 동시에 진행된다는 표현. 동일 job은 실행 중 interval을 놓치므로 “state는 먼저 break로 저장되며, 다음 Go 갱신은 helper 종료 뒤 재개”로 써야 한다.

## 5. 구현–근거 추적

| 구현 | 근거/AC | 실제 확인 | 잔여 불확실성 |
|---|---|---|---|
| 고정 ready 카드와 Start | 문제: 추천→실행 표면 단절, AC-01 | 코드·모델 unit·build PASS | primary/secondary 실제 배치, no-auto-start 육안 NOT RUN |
| 123초 guard | 전체 휴식 연장 방지, AC-02/04 | 122 거부·123 허용 unit PASS | 실제 버튼 disabled 전환과 카피 육안 NOT RUN |
| 120초 + 3초 상태 모델 | 명시적 완료, AC-02 | virtual 123 ticks PASS | 실제 Timer drift와 정확한 3초 표시 NOT RUN |
| Cancel→ready | 취소 가능성·강제 완화, AC-03 | cancel/restart unit PASS | 포인터·Space·focus 실제 동작 NOT RUN |
| Esc/elapsed Skip | 기존 회피 정책 보존, AC-03 | diff에서 handler/정책 확인 | phase별 실제 입력 NOT RUN |
| 전체 timeout 우선 | Go break가 상위 소유자, AC-04 | `main.swift:260-283` 소스 확인 | UI adapter 자동 테스트 없음 |
| 접근성 label/value/announcement | AC-06 | build PASS | VoiceOver 실제 발화·순서·3초 충분성 NOT RUN |
| no state/config/network | 프라이버시·TIDY-002 비악화, AC-07 | diff·source scan PASS | 없음(이번 변경 범위) |

## 6. 반증·실패 시나리오

### S-1 — 남은 전체 휴식 122초에서 시작 시도

- 재현: `swift test --filter GuidedBreakSessionTests`의 `testStartRequiresActivityAndCompletionBudget`.
- 실제 결과: `start(availableBreakSeconds: 122)`는 `false`, phase는 `.ready`; 123초에서는 `.running(120)`.
- 기대/판정: 가이드가 전체 휴식을 연장하지 않아야 한다. **PASS**.
- 영향 AC: AC-02, AC-04.

### S-2 — 진행 중 취소 후 재시작

- 재현: `testCancelReturnsRunningSessionToReadyAndAllowsRestart`.
- 실제 결과: 10 tick 후 cancel은 `.ready`로 복귀하고 남은 전체 휴식 170초에서 재시작 가능.
- 기대/판정: 취소가 overlay 전체 종료나 persisted state write가 아니어야 한다. **PASS**.
- 영향 AC: AC-03, AC-07.

### S-3 — overlay 실행 중 다음 60초 launchd interval

- 재현: 코드 경로 `runCheck → executeActions → breakscreen.Show → showOverlay → cmd.Run()`과 `man launchd.plist`의 `StartInterval` 규칙 대조.
- 실제 결과: helper가 실행 중이면 `check` 프로세스가 반환하지 않으며 해당 interval firing은 missed 된다.
- 기대/판정: “동시에 다음 Go tick이 돈다”는 설명은 **FAIL/반증**. 다만 state는 helper 실행 전에 `break`로 저장되어 schema 손상은 없다.
- 영향 AC: AC-07, 60초 실제 조건 문서화.
- 수정 후보: `docs/product/utility-mvp-prd.md`, `docs/product/utility-mvp-acceptance.md`, `docs/product/utility-mvp-implementation.md`; 제품 코드 변경 필요 여부는 PM/Planner가 결정한다.

### S-4 — VoiceOver가 완료 announcement를 끝까지 읽기 전에 3초 종료

- 재현: 실제 VoiceOver + 123초 runtime 필요.
- 실제 결과: **NOT RUN**. 코드상 high-priority announcement 요청 직후 완료 화면은 3초만 유지된다.
- 기대/판정: 이해 가능한 완료 피드백이어야 한다. **미검증이며 P0 통과 근거가 없음**.
- 영향 AC: AC-02, AC-06.
- 수정 후보: 수동 QA 결과에 따라 `main.swift` 완료 표시 시간 또는 P-009/AC 문서.

## 7. 알림 피로·선택 부담·방해·접근성·프라이버시

- **알림 피로**: 새 system notification은 추가하지 않았고 완료 announcement 1회만 코드에 있다. 실제 1회성은 수동 미검증이다.
- **선택 부담**: 복수 선택을 제거해 조작 선택지는 줄었다. 이것이 장기 무시를 줄인다는 효과는 미검증이다.
- **방해/다크 패턴**: 자동 시작, 확인 함정, 새 강제 입력, 취소 은닉은 없다. 기존 `.screenSaver` 전체화면 자체의 강한 방해는 유지된다. Start는 선택적이고 Cancel/Esc/Skip이 존재한다.
- **접근성**: 텍스트로 phase를 구분하고 새 애니메이션은 없다. 그러나 실제 VoiceOver, keyboard-only, Reduce Motion, Increase Contrast, multidisplay 검증이 없어 P0를 PASS로 판정할 수 없다.
- **프라이버시**: 고정 로컬 콘텐츠와 process-local phase만 추가됐다. 새 네트워크, 계정, 식별자, 로그 이벤트, state/config/history write는 없다.

## 8. 자동 품질 게이트 재실행

재검증 시각은 2026-08-19 10:52 KST다. 이전 구현자 로그를 복사하지 않고 현재 working tree에서 다시 실행했다.

| 명령 | 2026-08-19 10:52 KST 실제 결과 |
|---|---|
| `go test ./...` | PASS, exit 0 |
| `go test -race ./...` | PASS, exit 0 |
| `go vet ./...` | PASS, exit 0 |
| `(cd helpers && swift test)` | PASS, 101 tests / 0 failures, exit 0 |
| `make build` | PASS, release Swift helpers + Go binary, exit 0 |
| `git diff --check` | PASS, exit 0 |
| staged diff 확인 | 빈 목록 |

외부 원문 6건은 같은 날 각 URL을 `curl -fsSL`로 다시 내려받아 직접 인용 문자열을 대조했다. `grounded-citations` 원장에 6개 URL과 각 원문 인용이 등록돼 있으며, `sources.py verify docs/product/utility-mvp-factcheck.md --evidence`는 `citations OK`로 통과했다. 로컬 `man launchd.plist`의 `StartInterval` 절은 “job is running during an interval firing, that interval firing will likewise be missed”라고 명시하며, `cmd/break-reminder/check.go:72-85`와 `internal/breakscreen/overlay_darwin.go:14-44`의 blocking 호출 경로와 일치한다.

## 9. 보호 자산과 범위 드리프트

- 구현 관련 변경/신규: `helpers/Sources/HelperCore/GuidedBreakSession.swift`, `helpers/Tests/HelperCoreTests/GuidedBreakSessionTests.swift`, `helpers/Sources/BreakScreenApp/main.swift`, `README.md`, `CHANGELOG.md`, 제품 문서.
- Go source/state/config/CLI, MenuBarApp, DashboardApp, notify, TUI activity, VERSION, Formula, release workflow에는 이 구현 diff가 없다.
- `.analysis/**` tracked 3건과 `.analysis/.serena/root PNG` untracked 자산은 시작 전부터 존재한 보호 기준선이다. 이번 검토는 이를 수정·삭제·이동·스테이징하지 않았다.
- 새 dependency, image, font, animation, telemetry, permission은 없다.

## 10. 잔여 위험

1. **High — P0 수동 접근성·실 UX 증거 부재**: Return/Space, focus loop, VoiceOver value/announcement, 3초 이해 가능성, 대비, 다중 display를 실제 관찰하지 않았다.
2. **Medium — launchd 설명 불일치**: overlay 중 60초 tick은 동시 실행되지 않고 interval이 missed 된다. 123초 happy path 후 갱신 지연은 제한적이지만, 사용자가 ready를 무시해 긴 overlay를 유지하면 기존 gap 처리와 통계 정밀도 문제가 계속될 수 있다.
3. **Medium — 행동효과 미검증**: 기능 경로는 짧아졌지만 실제 start/completion율, 알림 피로, 반복 사용 유지율 데이터가 없다.
4. **Medium — 활동 최적성·안전성 미검증**: 고정 2분 목·어깨 활동은 제품 가정이며 개별 사용자의 신체 조건을 평가하지 않는다.
5. **Low — 문서 확정성**: README/CHANGELOG가 자동·소스 검증만 된 keyboard/accessibility 동작을 확정형으로 기술한다.

## 11. 증거 신뢰도

- **A**: 현재 코드/diff, 직접 재실행한 test/vet/build, 로컬 macOS man page, 공식 제품 기능 문서.
- **B**: 독립 메타분석의 제한된 일반화 가능한 방향.
- **C**: 조작 단계 감소가 행동 마찰 감소로 이어진다는 제품 해석.
- **D**: 실제 사용자 완료율, 최적 활동·길이, VoiceOver 실제 이해도, block-mode 도달률.
- **Conflicting**: “overlay 중 60초 Go tick”은 PRD/AC의 논리 검증 표현과 실제 launchd scheduling이 충돌한다. 실제 runtime 판단에는 macOS man page와 blocking 코드 경로를 우선한다.

## 12. 제품·디자인·기술 결정 영향

- PM/Planner는 이 구현을 “생산성/건강/완료율 개선이 검증된 기능”으로 승인하면 안 된다.
- 최종 품질 게이트 전에 QA-01~05 중 실제 UI·키보드·VoiceOver·123초 runtime을 사람이 수행하고 증거를 남겨야 한다.
- 60초 조건 문서는 “동시 tick”이 아니라 “state 선저장 + helper 종료 뒤 launchd 재개”로 수정해야 한다.
- 원격 telemetry를 추가해 증거 공백을 메우지 말고, 별도 동의 기반 사용자 관찰/인터뷰를 우선한다.

## 13. 최종 권고

**수정 후 진행 / 현재 판정 FAIL.**

자동 모델·회귀·build·privacy·scope 게이트는 통과했다. 그러나 완료 기준이 요구한 P0 실제 UX·접근성·123초 실런타임 증거가 없고, 60초 launchd 동시 실행 가정은 반증됐다. 따라서 이 문서 시점에는 전체 MVP를 PASS로 판정할 수 없다. PM의 최종 review는 최소한 수동 QA 결과와 launchd 문서 정정을 요구해야 한다.

아래 외부 원문 6건의 접근일은 모두 2026-08-19 KST다. 제목·URL·직접 인용은 인용 원장 `/tmp/utility-mvp-factcheck/ledger.json`에서 기계적으로 렌더링하고 `verify --evidence`로 대조했다.

## Sources

[1] https://pmc.ncbi.nlm.nih.gov/articles/PMC9432722 — Give me a break! systematic review and meta-analysis
    > "Overall, only four out of twenty-two studies were assessed with a low risk of bias, whereas one presented a high risk for at least half of the criteria. Thus, we are inclined to consider the risk of bias in our overall sample as being somewhat unclear."
    > "Overall, the data support the role of micro-breaks for well-being, while for performance, recovering from highly depleting tasks may need more than 10-minute breaks."
[2] https://pmc.ncbi.nlm.nih.gov/articles/PMC12164069 — Computer prompt software intervention: systematic review and meta-analysis
    > "From 17,880 records, 18 studies involving 1164 office workers were included in the analysis."
    > "Computer prompt software interventions show effectiveness in reducing sitting time among office workers. However, more long-term prospective studies with larger sample sizes are needed to accurately determine the effectiveness of computer prompts on various work- and health-related outcomes."
[3] https://github.com/hovancik/stretchly/blob/trunk/README.md — Stretchly README
    > "By default, there is a 20 second Mini break every 10 minutes and a 5 minute Long break every 30 minutes (after 2 Mini breaks)."
    > "When a break starts, you can postpone it once for 2 minutes (Mini breaks) or 5 minutes (Long breaks). Then, after a specific time interval passes, you can skip the break."
[4] https://breaktimer.app — BreakTimer official site
    > "BreakTimer lets you know when breaks are about to start, so you can quickly skip or snooze if timing is tight."
[5] https://lookaway.com — LookAway official site
    > "LookAway waits for the right moment to show a break reminder, and gives you a heads-up beforehand."
    > "Automatically pauses during Screen Recording Meetings or Calls Video Playback"
[6] https://developer.apple.com/design/human-interface-guidelines/notifications — Apple Human Interface Guidelines: Notifications
    > "A notification gives people timely, high-value information they can understand at a glance."
