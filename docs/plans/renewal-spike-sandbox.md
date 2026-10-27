# Phase 0 스파이크: App Sandbox 검증 결과

_작성일: 2026-10-20 · 작업지시서: [renewal-phase-0-foundation.md](renewal-phase-0-foundation.md) T0.C.1_

## 방법

linetta와 같은 방식으로 Mac App Store 인증서 없이 **Developer ID + `app-sandbox` 엔타이틀먼트**로 로컬 샌드박스 빌드를 만들어 확인했다.

```bash
TIMESTAMP=0 scripts/build-app.sh --sandbox --spike   # bin/Break Reminder (Sandbox).app
log stream --style compact --predicate 'sender == "Sandbox" OR eventMessage CONTAINS "deny("' &
open -n --env BREAK_REMINDER_SPIKE=1 "bin/Break Reminder (Sandbox).app"
cat ~/Library/Containers/com.devlikebear.BreakReminder/Data/Library/Application\ Support/BreakReminder/spike-report.json
```

- 앱: `helpers/Resources/app-sandbox.entitlements` (app-sandbox, network.client/server, calendars, user-selected.read-write)
- 사이드카(`engine`, `break-screen`, `sandbox-probe`): `engine-sandbox.entitlements` (app-sandbox + inherit만)
- `BREAK_REMINDER_SPIKE=1`이면 앱이 `SpikeProbe`(Swift)를 돌리고, 번들에 넣은 `tools/sandbox-probe`(Go, 엔진과 같은 상속 샌드박스)로 사이드카 쪽 항목을 확인한 뒤 종료한다.
- 확인 환경: macOS 27.0.1, 단일 모니터, Apple Silicon

## 결과

| 항목 | 결과 | 근거 |
|---|---|---|
| 사이드카 실행 (stdio JSON-RPC) | **PASS** | 샌드박스 앱이 `engine --stdio`를 띄우고 상태 이벤트 수신(`engine_running`, `engine_status_received` = true). 엔진 프로세스 위반 로그 0건 |
| 컨테이너 경로 쓰기 | **PASS** | 앱이 `BREAK_REMINDER_HOME=~/Library/Containers/com.devlikebear.BreakReminder/Data/Library/Application Support/BreakReminder`를 넘기고, 엔진이 `state`, 로그, time-tools 파일을 그 아래 생성. 샌드박스 안에서는 `HOME`도 컨테이너로 바뀜 |
| 컨테이너 밖 쓰기 (`~/.break-reminder-*`, `~/Library/LaunchAgents`) | 차단 (예상대로) | `deny(1) file-write-create` — 기존 launchd 설치 경로는 앱스토어판에서 쓸 수 없음을 확인 |
| `CGEventSource` 유휴 시간 | **PASS** | 샌드박스 앱에서 `idle_seconds=252` 정상값 |
| 휴식 화면(`break-screen`) 실행 | **PASS** (단일 모니터) | 상속 샌드박스로 3초 오버레이 실행 후 `exit 0`. AppKit의 `system-info vfs.disk-space` 거부 로그만 있음(프레임워크 소음, 동작 영향 없음) |
| 전체화면 오버레이 멀티모니터 | **needs user check** | 테스트 환경이 모니터 1대 |
| 127.0.0.1 HTTP 서버 (P2 외부 MCP) | **PASS** | 사이드카(상속)에서 `127.0.0.1:0` listen + GET 왕복 `pong`. `network.server`는 앱 엔타이틀먼트로 충분 |
| 다른 프로세스 CPU/메모리 (P3 진단) | **FAIL** | `proc_pid_rusage` → `deny(1) process-info-rusage`(같은 샌드박스의 부모 앱조차 거부), `/bin/ps` → `forbidden-exec-sugid`. 프로세스 목록 자체(`sysctl kern.proc.all`)는 PASS(583개) |
| EventKit 권한 요청·유지 | **needs user check** | 요청 전 상태 `notDetermined`(0). 권한 다이얼로그 클릭이 필요해 자동 확인 불가 |
| `SMAppService` 로그인 항목 | **needs user check** | `bin/`에서 실행 시 상태 `notFound`(3). `/Applications`에 설치된 앱에서 등록해야 의미 있음. 사용자 로그인 항목을 바꾸므로 자동으로 하지 않음 |
| Keychain (T0.D.1 결정용) | 미실행 | 작업 지침상 키체인 항목 쓰기 금지. 아래 결정은 구조적 근거로 내림 |

참고로 사이드카에서 시스템 명령 실행은 setuid가 아니면 허용됐다: `ioreg`, `say -o <컨테이너 파일>`, `osascript -e 'return 1'`, `defaults read -g AppleLanguages` 모두 PASS. 다만 앱스토어 심사 관점에서 외부 명령 의존은 줄이는 방향을 유지한다(엔진은 이미 OS 호출을 앱에 넘김).

## needs user check — 직접 확인 절차

1. **EventKit 권한 요청·유지**
   ```bash
   open -n --env BREAK_REMINDER_SPIKE=1 --env BREAK_REMINDER_SPIKE_CALENDAR=1 "bin/Break Reminder (Sandbox).app"
   ```
   권한 다이얼로그에서 "허용" → `spike-report.json`의 `calendar_request`가 `granted`, `calendar_events_next_24h`가 숫자인지 확인. 이어서 `TIMESTAMP=0 scripts/build-app.sh --sandbox --spike`로 **재빌드**하고, 로그아웃/로그인 후 같은 명령을 다시 실행해 다이얼로그 없이 `calendar_status_before`가 `3`(fullAccess)인지 확인.
2. **멀티모니터 오버레이**: 외부 모니터를 연결한 상태에서 1번 첫 줄 명령(캘린더 변수 없이)을 실행 → 3초 동안 모든 화면에 휴식 화면이 뜨는지, `break_screen`에 `screens=2`가 찍히는지 확인.
3. **로그인 항목**: `bin/Break Reminder (Sandbox).app`을 `/Applications`로 복사 → 실행 → 오른쪽 클릭 메뉴 "로그인 시 실행" 체크 → 시스템 설정 > 일반 > 로그인 항목에 표시되는지, 재로그인 후 자동 실행되는지 확인 후 체크 해제.

검증이 끝나면 정리: `rm -rf ~/Library/Containers/com.devlikebear.BreakReminder` (스파이크 데이터만 있음).

## 결정과 로드맵 반영

- **T0.D.1 Keychain**: Go `internal/secrets`를 Security.framework **cgo**(SecItem API)로 구현한다. 근거: 직접 배포판은 앱이 꺼져 있어도 launchd `check`·CLI가 TTS 키를 읽어야 하므로 Go가 키체인을 직접 읽어야 한다. 앱스토어판은 사이드카의 키체인 접근이 불확실하므로 Swift가 SecItem으로 저장하고 엔진에는 RPC(`secrets.set`)로 메모리에만 전달한다. 두 경로 모두 같은 `secrets.Store` 인터페이스 뒤에 둔다.
- **P3 진단("왜 느리지?")**: 앱스토어판은 다른 프로세스별 CPU/메모리를 볼 수 없다. 앱스토어판은 시스템 전체 지표(`host_statistics64`, 메모리 압력, 디스크 여유)와 "어느 앱이 전면에 있었는지"(NSWorkspace) 결합으로 범위를 줄이고, 프로세스별 상위 목록은 직접 배포판 전용 기능으로 표기한다. → `renewal-phase-3-signals.md` 수정 필요.
- **P2 외부 MCP**: 127.0.0.1 HTTP 서버가 사이드카에서 동작하므로 계획대로 진행.
- **P4 launchd 퇴역**: 컨테이너 밖 쓰기·LaunchAgents 쓰기가 막힘을 재확인. 앱스토어판에서 v0.16 데이터 자동 이전은 불가하므로, 직접 배포판에서만 이전하고 앱스토어판은 새로 시작(또는 사용자가 폴더를 한 번 선택해 가져오기)으로 한다. → `renewal-phase-4-rebrand.md`, `renewal-phase-5-app-store.md` 반영 필요.
- **언어 감지**: 엔진은 `defaults` 실행에 기대지 않고 앱이 `BREAK_REMINDER_LANGUAGE`를 넘긴다(이미 구현).
