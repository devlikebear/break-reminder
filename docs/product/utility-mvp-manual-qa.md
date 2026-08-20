# Start One Guided Break — 수동 QA 기록

- 실행 시각: 2026-08-19 11:00–11:03 KST; 권한 부여 후 재시도 14:38–14:57 KST
- 실행 환경: macOS 26.5.2 (Build 25F84), Apple M4
- 디스플레이: `화면 공유 가상 디스플레이` 1대, 2560×1440 @ 30Hz, main/mirror off. 재시도에서는 `Online: Yes`이고 `screencapture`가 성공했다.
- VoiceOver: 프로세스 미실행
- Reduce Motion / Increase Contrast: `com.apple.universalaccess`에 해당 preference key 없음
- 판정: **PARTIAL PASS / BLOCKED — read-only 화면·AX 관찰은 가능해졌으나 headless 승인 정책이 CuaDriver 입력을 차단해 필수 상호작용·VoiceOver·123초 실런타임은 계속 NOT RUN**

## 사전 빌드

명령:

```bash
make build
```

실제 결과: PASS. Swift release helper와 Go binary가 생성됐고 `bin/break-screen`이 존재한다.

## 실행 및 관찰 가능성 확인

### 600초 ready fixture 실행

명령:

```bash
bin/break-screen --duration 600 --skip-after 120 --work-min 60 --break-min 10
```

실제 결과:

- `BreakScreenApp` 프로세스는 실행 상태를 유지했다.
- 그러나 유일한 가상 디스플레이가 asleep 상태여서 `screencapture -x /tmp/guided-ready.png`가 `could not create image from display`로 실패했다.
- `caffeinate -u -t 2` 후 재시도해도 동일하게 실패했다.
- `AXIsProcessTrusted()`는 `false`였다. System Events로 `BreakScreenApp` 접근성 트리를 읽는 시도는 60초 후 timeout됐다.
- fixture 프로세스는 QA가 불가능함을 확인한 뒤 종료했다. 화면의 ready 카드, no-auto-start, focus, 카피를 관찰했다고 주장하지 않는다.

### GUI 자동화 도구 상태

명령:

```bash
hermes computer-use doctor
hermes computer-use install
```

실제 결과:

- 최초 doctor: `cua-driver: not installed`.
- 설치: exit 0.
- 설치 후 안내된 macOS Accessibility와 Screen Recording 권한은 현재 headless Kanban run에서 승인할 수 없다.
- 현재 Hermes 세션은 시작 시 computer-use MCP를 로드하지 않았으므로 설치 후에도 세션 내 GUI 캡처/입력을 사용할 수 없다. 새 Hermes 세션과 사용자 권한 승인이 필요하다.

## 시나리오 결과

| QA | 범위 | 결과 | 실제 관찰 |
|---|---|---|---|
| QA-01 | 600초 ready, 122/123 guard, no-auto-start, primary/secondary | BLOCKED / NOT RUN | 프로세스 실행만 확인. 화면 캡처·AX 접근 불가 |
| QA-02 | stopwatch 60/120/123초, 완료, 3초 자동 종료 | BLOCKED / NOT RUN | Start를 실제 UI로 실행·관찰할 입력 경로 없음 |
| QA-03 | Cancel→ready→restart, phase별 Esc, 5초 전후 Skip/focus | BLOCKED / NOT RUN | 실제 focus/키 입력 검증 불가 |
| QA-04 | Return/Space/Tab/Shift-Tab, VoiceOver label/value/hint, announcement 1회 | BLOCKED / NOT RUN | VoiceOver off, AX 권한 없음, 오디오/화면 관찰 불가 |
| QA-05 | Reduce Motion, Increase Contrast, focus ring, multi-display | BLOCKED / NOT RUN | 접근성 설정 변경·시각 확인 불가; 디스플레이도 1대뿐 |
| QA-06 | helper restart, state 선저장, helper 종료 뒤 launchd interval 재개 | BLOCKED / NOT RUN | GUI phase와 실제 launchd 재개를 확인할 수 없어 실관찰 증거 생성 불가 |

## 차단 해제 조건

1. 로그인된 실제 macOS 데스크톱 또는 awake 상태의 캡처 가능한 디스플레이.
2. Hermes/터미널(또는 CuaDriver)에 macOS Accessibility와 Screen Recording 권한 부여.
3. `cua-driver` MCP가 로드된 새 Hermes 세션 시작.
4. VoiceOver 실제 음성 낭독을 들을 수 있는 작업자 또는 녹음 가능한 오디오 경로.
5. multi-display 항목을 PASS로 판정하려면 실제 두 번째 디스플레이. 한 대뿐이면 해당 관찰은 `N/A (환경 미충족)`로 분리한다.

## 제품 판정 영향

아래 내용은 첫 시도의 판정이며, 권한 부여 후 재시도 결과가 최신 상태다. AC-01/03/04/06의 일부 read-only 관찰은 수행했지만 AC-02의 123초 실런타임, AC-06의 입력·VoiceOver, AC-07의 launchd 실런타임은 계속 `NOT RUN`이다. 자동 빌드 PASS만으로 미실행 항목을 PASS로 승격하지 않는다.

## 권한 부여 후 재시도 — 2026-08-19 14:38–14:57 KST

### 환경과 명령

- OS: macOS 26.5.2 (Build 25F84), Apple M4.
- Display: 화면 공유 가상 디스플레이 1대, 2560×1440 @ 30Hz, main/online, mirror off. 실제 두 번째 display는 없어 multi-display는 `N/A`다.
- `hermes computer-use doctor`: cua-driver 0.20.0, MCP active, bundle `com.trycua.driver`, Accessibility granted, Screen Recording granted, AX trusted/reachable.
- VoiceOver: `pgrep -fl VoiceOver` 결과 없음. `reduceMotion`, `increaseContrast` preference key도 현재 domain에 없었다.
- 빌드: `make build` exit 0. release Swift helper와 Go binary를 다시 생성했다.
- 실행 fixture: `600/120/work 60/break 10`, `122/120`, `180/5`, `124/120`을 실제 `bin/break-screen`으로 실행했다.
- 화면 증거: `screencapture -x`가 성공했고, CuaDriver를 PID/window-id에 exact bind해 full-screen window와 AXButton을 read-only로 읽었다.

### 실제 관찰

1. **600초 ready / no-auto-start — PASS(관찰 범위).** `휴식 시간이에요`, 전체 `MM:SS`, 진행 막대, `2분 가이드`, `2분 동안 서서 목과 어깨를 풀어보세요`, `같은 화면에서 천천히 따라 해요.`, 파란 `시작`, 통계 `오늘: 작업 1h · 휴식 10m`, disabled Skip, Esc 안내가 한 화면에 잘림 없이 보였다. 사용 입력 없이 약 9분 동안 ready가 유지돼 자동 시작하지 않았다. CuaDriver exact AX에는 enabled `시작` button이 노출됐다.
2. **짧은 휴식 — PASS(관찰 범위).** 122초 fixture가 경과해 `01:55`일 때 `이번 휴식에는 2분이 남지 않았어요.`와 disabled `시작`이 보였다. 180초 fixture에서는 일반 ready 안내와 enabled `시작`이 보였다. 도구 왕복 지연 때문에 화면이 정확히 `02:03`일 때의 한 프레임은 확보하지 못해 123초 UI 경계 자체는 `NOT RUN`으로 남긴다.
3. **Skip — PARTIAL.** 180초/skip-after 5 fixture를 read-only capture했을 때 `Skip Break` enabled 표면과 AXButton 노출을 확인했다. 정확한 5초 전/후 focus 전환은 입력·연속 캡처 승인 부재로 `NOT RUN`이다.
4. **레이아웃/대비 — PASS(현재 기본 설정).** 2560×1440에서 카드·타이머·문구가 잘리지 않았고 title/전체 시간/card title/instruction의 텍스트 위계가 분명했다. 화면 상태는 색만으로 전달되지 않았다. 다만 focus ring, Increase Contrast on, Reduce Motion on은 설정 전환을 실행하지 못해 `NOT RUN`이다.
5. **입력/VoiceOver/123초 — BLOCKED / NOT RUN.** CuaDriver read-only capture는 성공했지만 `click`은 `approval prompt timed out — the user did not respond. Silence is not consent; do not retry without the user.`로 거부됐다. headless Kanban run에서 승인할 사용자가 없어 Start/Cancel/Return/Space/Tab/Shift-Tab/Esc/Skip을 수행할 수 없었다. 따라서 running/completed, stopwatch 60/120/123초, 완료 announcement 1회와 3초 이해 가능성을 관찰하지 않았다.
6. **권한 dialog 안전.** CuaDriver 관련 macOS privacy dialog가 다른 앱 뒤에 보였지만 자동으로 클릭하지 않았다. doctor의 granted 결과와 별개로 OS permission UI를 우회하지 않았다.

### 재시도 판정표

| QA | 결과 | 실제 관찰 / 잔여 |
|---|---|---|
| QA-01 | PARTIAL PASS | 600초 ready/no-auto-start와 short-state 시각 PASS; 정확한 123초 한 프레임·secondary display N/A |
| QA-02 | BLOCKED / NOT RUN | Start 입력 승인이 없어 60/120/123초 running/completed 관찰 불가 |
| QA-03 | BLOCKED / NOT RUN | Cancel→ready→restart, phase별 Esc, 정확한 5초 focus 전환 불가 |
| QA-04 | BLOCKED / NOT RUN | keyboard mutation 승인과 실제 VoiceOver 음성 경로 없음 |
| QA-05 | PARTIAL PASS | 기본 설정 시 텍스트/버튼 대비·잘림 PASS; Reduce Motion/Increase Contrast on과 focus ring NOT RUN; multi-display N/A |
| QA-06 | BLOCKED / NOT RUN | helper fresh-ready의 화면은 재실행마다 관찰했으나 running 강제 종료와 실제 launchd interval 재개는 수행하지 않음 |

### 실패 기록과 수정 후보

- 증상: read-only AppKit capture는 되지만 모든 CuaDriver click/key mutation이 승인 timeout으로 거부된다.
- 재현: exact PID/window bind 후 `computer_use click(element=시작)`.
- 기대: headless QA run에 사전 승인된 bounded mutation capability가 있어 지정된 break-screen window에만 입력할 수 있어야 한다.
- 영향 AC: AC-02, AC-03, AC-04의 정확한 123초 UI, AC-06, AC-07의 multi-display/launchd 수동 범위.
- 수정 후보: 제품 코드 결함은 관찰되지 않아 없음. QA 인프라/실행 정책에서 break-screen PID/window에 한정한 CuaDriver mutation 승인과 실제 VoiceOver 청취 경로가 필요하다.

## 명시적 `--yolo` Guided Break 재검증 — 2026-08-20 14:43–15:09 KST

### 범위·환경·사전 조건

- 사용자 승인 범위: 이 1회 QA에서 `/Users/changheonshin/workspace/myworks/break-reminder/bin/break-screen` fixture와 CuaDriver로 확인한 각 `break-screen`의 정확한 PID/window만 캡처·조작했다. 브라우저·계정·네트워크·권한 UI·시스템 설정은 조작하지 않았다.
- OS/디스플레이: macOS 26.5.2 arm64, 캡처 가능한 2560×1440 단일 display. secondary display는 없어 multi-display는 `N/A`다.
- `hermes computer-use doctor`: PASS — cua-driver 0.20.0, MCP active, `com.trycua.driver`, Accessibility/Screen Recording granted, AX trusted/reachable. Direct ScreenCaptureKit probe는 read-only doctor 특성상 생략됨.
- `make build`: PASS — Swift release helper 3개와 Go binary 빌드 완료. 빌드 뒤 보호 대상 tracked diff가 새로 생기지 않았다.
- CuaDriver는 각 fixture의 CoreGraphics window number를 PID로 조회한 뒤 `(pid, window_id)` exact match가 확인된 창만 사용했다. mutation 뒤에는 CuaDriver AX read-back 또는 프로세스 exit를 다시 확인했고 성공한 입력은 반복하지 않았다.

### 실행 fixture와 실제 결과

| 시나리오 | 명령/조작 | 실제 관찰 | 판정 |
|---|---|---|---|
| 600초 ready/no-auto-start | `bin/break-screen --duration 600 --skip-after 120 --work-min 60 --break-min 10`; 입력 없이 기존 9분 관찰 증거 재사용 + 이번 실행 fresh capture | `09:45`, 고정 2분 가이드, enabled `시작`, disabled Skip, 통계와 Esc 안내. 이번 실행도 입력 전 ready 유지 | PASS |
| 122 guard | `--duration 122 --skip-after 120`; Cua AX tree | 전체 `01:47`, `이번 휴식에는 2분이 남지 않았어요.`, `시작`과 `Skip (available in 2min)`에 AX press action 없음 | PASS |
| 123 guard | `--duration 124`에서 초기 `02:04`/enabled Start를 읽고 1.05초 뒤 exact PID/window에 Return; 다음 AX read-back | 전체 `02:02`인데 이미 running `01:59`와 `가이드 취소`가 노출됐다. 122에서는 시작할 수 없으므로 경계 tick의 123초에서 시작이 수락된 실제 결과다 | PASS |
| Start 클릭 | 600초 fixture에서 Start 좌표를 Cua AX hit-test로 1회 클릭 | ready `시작`이 running `가이드 취소`로 바뀌고 `01:59`/첫 단계가 노출됨 | PASS (즉시 read-back은 첫 global tick 뒤 `01:59`; `02:00` 단일 프레임은 미포착) |
| Return/Space | ready에서 Return, Cancel 후 fresh ready에서 Space 각각 1회 | 두 입력 모두 running으로 전환; Space 재시작 read-back `01:58` | PASS |
| 약 60초 | 600초 fixture를 Return으로 시작하고 Cua AX state를 bounded poll | guide `01:00`, 전체 `09:00`, 단계 `고개를 천천히 좌우로 기울이고, 통증이 있으면 멈추세요.` exact read-back | PASS |
| 120초 completed/3초 종료 | 같은 fixture에서 completed를 bounded poll하고 프로세스 종료 epoch 기록 | `완료했어요`, `편안하게 남은 휴식을 이어가세요.`, `3초 후 이 화면을 닫아요.`; completed 관찰 epoch 1787205941.85904, exit 1787205944.6247108, 차이 약 2.766초, exit 0 | PASS |
| Cancel→ready→restart | running의 `가이드 취소`를 Cua AX hit-test 클릭, ready read-back 후 Space | ready의 `시작`/고정 안내로 복귀하고 Space로 running 재진입 | PASS |
| ready/running/completed Esc | 각 phase의 exact PID/window에 Escape 1회씩 전송 | ready와 running은 exit 0. completed는 completed AX read-back 직후 Esc→0.141초 내 exit 0 | PASS |
| Skip 전/후 | 기존 600/120 disabled 증거 및 이번 122 AX read-back 재사용; `--duration 180 --skip-after 5`에서 enabled Skip read-back 후 1회 클릭 | 전: disabled/AX action 없음. 후: enabled `Skip Break`/AX press action 있음; 클릭 후 exit 0 | PASS |
| Tab/Shift-Tab | ready에서 Tab 1회, Shift-Tab 1회 후 Return | CuaDriver가 exact PID에 두 key delivery를 보고했고 AX상 action control은 Start→Skip 두 개뿐이며, 이어진 Return은 Start를 실행했다. 다만 Cua AX payload가 `AXFocused`를 반환하지 않고 screenshot에서도 focus ring을 확정할 수 없음 | PARTIAL PASS — key delivery/loop 동작 PASS, focus ring·각 단계 focused 속성은 N/A(관찰 API 한계) |
| AX label/value/help | ready/running/completed exact AX tree | Start: label `2분 목과 어깨 스트레칭 시작`, help `같은 화면에서 2분 가이드를 시작합니다.`; running: title/`01:00`/단계; Cancel help `가이드를 멈추고 휴식 화면으로 돌아갑니다.`; completed title/value/help 노출 | PASS |
| announcement/실제 음성 | completed 화면과 소스의 `NSAccessibility.post(.announcementRequested)` API 범위만 확인 | 실제 VoiceOver를 켜지 않았고 오디오를 청취하지 않았다. CuaDriver는 announcement event stream/count를 노출하지 않음 | N/A — 실제 낭독·1회 event 관찰 불가 |
| Reduce Motion/Increase Contrast | 시스템 설정을 변경하지 않음 | 현재 기본 화면만 관찰. 설정 전환은 승인 범위 밖/환경 미충족 | N/A |
| multi-display | 단일 display | secondary 표면 미실행 | N/A |
| launchd 실런타임 | LaunchAgent를 kick/수정하지 않음 | 기존 source 계약(`cmd.Run()` blocking, running 중 StartInterval missed, helper 종료 뒤 다음 interval 재개)과 기존 자동 증거를 재사용. 실제 LaunchAgent firing은 이번 좁은 fixture/Cua 범위 밖 | NOT RUN (source/자동 계약 PASS 유지) |

### 관찰상 결함·잔여

- **제품 기능 결함은 발견하지 못했다.** Start/Cancel/restart/Skip/Esc와 120초 completion/자동 종료는 실제 fixture에서 동작했다.
- 정확한 `02:00` 한 프레임은 fixture launch 후 window discovery/첫 global 1초 tick과 read-back이 경합해 확보하지 못했다. 즉시 AX read-back은 `01:59`였고 이후 `01:00`, completed, 약 3초 종료는 연속 실제 런타임으로 확인했다. 수정 후보가 필요한 명확한 제품 결함으로 판정하지 않는다.
- focus ring 및 `AXFocused` 속성, accessibility announcement event 횟수, 실제 VoiceOver 음성은 현재 CuaDriver 관찰 표면에 없어 정직하게 N/A/부분 판정으로 남긴다.
- 모든 fixture 프로세스는 Skip/Esc/자동 종료 또는 QA harness kill로 시나리오 뒤 종료했다.
