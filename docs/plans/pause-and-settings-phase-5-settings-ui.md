# Phase 5: Settings UI 탭 — 작업지시서

_작성일: 2026-05-12_
_속한 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)_
_예상 소요: 4-5시간_

## 페이즈 목표

대시보드 탭바에 "설정" 탭이 추가되고, 사용자가 핵심 8-10개 설정을 폼으로 편집해서 저장할 수 있다. 저장은 Phase 4의 `break-reminder config set` CLI를 호출해서 검증/원자성 보장. 저장 실패 시 GUI에 에러 메시지 표시.

## 전제 조건

- [ ] Phase 4 완료 및 사용자 승인
- [ ] `break-reminder config set` CLI 동작 확인
- [ ] HelperCore `AppConfig`가 핵심 필드 모두 노출

## 포함 기능

1. `DashboardTab`에 `.settings` 케이스 추가
2. `SettingsTabView` 신규 — 폼 기반 편집 UI
3. 노출할 핵심 필드 (8-10개):
   - 작업 시간 (work_duration_min) — Stepper 또는 Slider
   - 휴식 시간 (break_duration_min) — Stepper
   - Idle 임계값 (idle_threshold_sec) — Stepper
   - Natural break 임계값 (natural_break_sec) — Stepper
   - 근무 시작/종료 시간 (work_start_hour/minute, work_end_hour/minute) — TimePicker (2개)
   - 근무 요일 (work_days) — Toggle 7개 (월-일)
   - Break screen 모드 (break_screen_mode) — Picker ("ask"/"block"/"notify")
   - 알림 활성 (notifications_enabled) — Toggle
   - TTS 활성 (tts_enabled) — Toggle
   - 휴식 활동 활성 (break_activities_enabled) — Toggle
   - 테마 (theme) — Picker ("auto"/"dark"/"light")
4. 저장 / 취소 / 기본값 복원 버튼
5. 변경 감지(dirty flag) — 저장 안 한 변경 있으면 저장 버튼 활성화, 다른 탭 이동 시 확인 (선택)
6. 에러 표시 — 저장 실패 시 alert

## 이 페이즈에서 하지 않는 것

- TTS engine / TTS API key / TTS Python cmd 같은 고급 옵션 → Out of Scope (yaml 직접 편집 유지)
- AI CLI / AI enabled → Out of Scope
- max_log_lines, check_interval_sec → Out of Scope (자주 안 바뀜, 고급)
- 다국어 — UI 텍스트는 한국어 + 영어 키 병기 OK

## 작업 체크리스트

### 작업 그룹 A: 탭 구조

- [ ] **T5.A.1** — `DashboardTab` enum에 `.settings` 추가
  - 파일: `helpers/Sources/DashboardApp/DashboardViewModel.swift`
  - 내용:
    ```swift
    enum DashboardTab: String, CaseIterable, Identifiable {
        case timer = "타이머"
        case stats = "통계"
        case insights = "인사이트"
        case settings = "설정"
        var id: String { rawValue }
    }
    ```
  - 검증: 컴파일 통과

- [ ] **T5.A.2** — `TabBarView`에 탭 항목 추가
  - 파일: `helpers/Sources/DashboardApp/TabBarView.swift`
  - 내용: 기존 탭 ForEach에 .settings가 자연스럽게 포함됨 (CaseIterable이므로 별도 작업 없을 수 있음). SF Symbol 아이콘 매핑이 있다면 `case .settings: return "gear"` 추가
  - 검증: 빌드 + 실행 시 탭 4개 표시

- [ ] **T5.A.3** — `DashboardAppMain`의 탭 컨텐츠 매핑
  - 파일: `helpers/Sources/DashboardApp/DashboardAppMain.swift`
  - 내용: 탭 선택에 따라 view를 swap하는 switch가 어디 있는지 확인하고 `.settings` 케이스에 `SettingsTabView(vm: vm)` 매핑
  - 검증: 설정 탭 클릭 시 빈 화면이라도 SettingsTabView 진입

### 작업 그룹 B: 폼 상태 모델

- [ ] **T5.B.1** — `SettingsFormState` 신설
  - 파일: `helpers/Sources/DashboardApp/SettingsTabView.swift` (신규)
  - 내용:
    ```swift
    @Observable final class SettingsFormState {
        var workDurationMin: Int
        var breakDurationMin: Int
        var idleThresholdSec: Int
        var naturalBreakSec: Int
        var workStartHour: Int
        var workStartMinute: Int
        var workEndHour: Int
        var workEndMinute: Int
        var workDays: Set<Int>
        var breakScreenMode: String
        var notificationsEnabled: Bool
        var ttsEnabled: Bool
        var breakActivitiesEnabled: Bool
        var theme: String

        var original: AppConfig // 초기값 보존 (dirty 체크용)

        init(from cfg: AppConfig) {
            self.workDurationMin = cfg.workDurationMin
            // ... 나머지 필드 초기화
            self.workDays = Set(cfg.workDays)
            self.original = cfg
        }

        var isDirty: Bool {
            // 각 필드를 original과 비교
            ...
        }

        func toChanges() -> [String: String] {
            // CLI args 만들기: "key=value" 형태
            // dirty 필드만 포함하면 더 효율적이지만, 일단 전부 보내도 됨
            ...
        }

        func reset() {
            // original로 복원
        }
    }
    ```
    - macOS 13 미만 타겟이면 `@Observable` 대신 `ObservableObject` + `@Published` 사용. 프로젝트 타겟 확인 (helpers/Package.swift).
  - 의존: Phase 4의 `AppConfig` 확장
  - 검증: 빌드 통과

- [ ] **T5.B.2** — `toChanges()` → CLI 인자 변환
  - 파일: `SettingsTabView.swift`
  - 내용:
    - bool: `"notifications_enabled=true"` 같은 문자열
    - work_days Set<Int> → 정렬해서 `"work_days=[1,2,3,4,5]"` 형식 (yaml flow style)
    - time picker → `work_start_hour=9 work_start_minute=0` 두 개로
    - 전체 키 매핑 정확히
  - 검증: 단위 테스트 또는 print 디버깅

### 작업 그룹 C: SettingsTabView UI

- [ ] **T5.C.1** — 폼 레이아웃
  - 파일: `helpers/Sources/DashboardApp/SettingsTabView.swift`
  - 내용:
    - SwiftUI `Form` + `Section`으로 그룹화:
      - **Section "타이머"**: workDurationMin(Stepper), breakDurationMin(Stepper), idleThresholdSec(Stepper), naturalBreakSec(Stepper)
      - **Section "근무 시간"**: workStart(DatePicker.hourAndMinute), workEnd(DatePicker.hourAndMinute), workDays(7개 Toggle 가로 배치)
      - **Section "동작"**: breakScreenMode(Picker), notificationsEnabled(Toggle), ttsEnabled(Toggle), breakActivitiesEnabled(Toggle)
      - **Section "테마"**: theme(Picker)
    - 각 필드 라벨: 한국어 + 영어 키 ("작업 시간 (work_duration_min)")
    - Form 하단: Save / Cancel / Reset to defaults 버튼 HStack
  - 참조: macOS SwiftUI Form은 List 스타일. settings 윈도우의 자연스러운 레이아웃.
  - 검증: 빌드 + 실행 시 모든 컨트롤 표시

- [ ] **T5.C.2** — 저장 액션
  - 파일: `SettingsTabView.swift` + `DashboardViewModel.swift`
  - 내용:
    - ViewModel에 `saveSettings(_ changes: [String: String]) -> Result<Void, Error>` 추가
      ```swift
      func saveSettings(_ changes: [String: String]) -> Result<Void, Error> {
          guard let cli = findHelper("break-reminder") else {
              return .failure(NSError(...))
          }
          let args = ["config", "set"] + changes.map { "\($0.key)=\($0.value)" }
          let p = Process()
          p.launchPath = cli
          p.arguments = args
          let errPipe = Pipe()
          p.standardError = errPipe
          do {
              try p.run()
              p.waitUntilExit()
              if p.terminationStatus != 0 {
                  let errMsg = String(data: errPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? "Unknown error"
                  return .failure(NSError(domain: "ConfigSave", code: Int(p.terminationStatus), userInfo: [NSLocalizedDescriptionKey: errMsg]))
              }
              refresh() // config 다시 읽기
              return .success(())
          } catch {
              return .failure(error)
          }
      }
      ```
    - View에서 Save 버튼 클릭 시:
      ```swift
      switch vm.saveSettings(form.toChanges()) {
      case .success: form.original = vm.config // 새 baseline
      case .failure(let err): self.errorMessage = err.localizedDescription; self.showError = true
      }
      ```
    - `@State var showError = false`, `@State var errorMessage = ""`
    - `.alert("저장 실패", isPresented: $showError) { Button("확인") {} } message: { Text(errorMessage) }`
  - 검증: 수동 — 잘못된 값(예: workStartHour=25은 UI에서 막혀 있으므로 다른 방식으로 검증)으로 저장 시도 시 에러 alert

- [ ] **T5.C.3** — Cancel / Reset 버튼
  - 파일: `SettingsTabView.swift`
  - 내용:
    - Cancel: `form.reset()` — original로 복원
    - Reset to defaults: AppConfig() 새 인스턴스로 초기화 (yaml은 기본값으로 다시 쓰기 위해 모든 필드를 set)
      - 또는 더 단순하게: `break-reminder config edit` 안내 메시지 표시 (yaml에서 모든 키 제거하면 다음 Load가 기본값 사용)
      - 권장: Reset 버튼은 UI 상태만 초기화. 진짜 yaml 초기화는 Out of Scope.
  - 검증: 수동 — Cancel 후 모든 필드가 원래 값으로 복귀

### 작업 그룹 D: 마무리

- [ ] **T5.D.1** — Dirty 체크 + Save 버튼 비활성화
  - 파일: `SettingsTabView.swift`
  - 내용:
    - Save 버튼 `.disabled(!form.isDirty)` 적용
    - (선택) 다른 탭 이동 시 dirty면 confirmation alert
  - 검증: 수동 — 값 안 바꾸면 Save 버튼 회색

- [ ] **T5.D.2** — refresh 후 폼 상태 동기화
  - 파일: `SettingsTabView.swift`
  - 내용:
    - View에서 `.onAppear`에 `form = SettingsFormState(from: vm.config)` 호출 (또는 init에서)
    - vm.config가 외부에서 바뀌었을 때(예: 다른 프로세스가 yaml 수정) 화면 갱신은 다음 onAppear까지 보류 — UX 단순성 우선
  - 검증: 수동 — 설정 탭 들어갔다 다른 탭 다녀와도 폼 상태 안 깨짐

- [ ] **T5.D.3** — README / CHANGELOG 업데이트
  - 파일: `README.md`, `CHANGELOG.md`
  - 내용:
    - CHANGELOG에 신규 섹션:
      ```
      ## [Unreleased]
      ### Added
      - GUI pause modes (Meeting/Focus/AFK) with auto-resume timer
      - Settings panel in dashboard for core configuration
      - CLI: `break-reminder pause --mode=... --duration=...`
      - CLI: `break-reminder config set key=value`
      ```
    - README "Usage" 또는 "Dashboard" 섹션에 새 기능 한 단락
  - 검증: 문서 렌더링 확인

---

## ✅ Phase 5 Checkpoint

**구현 확인:**
- [ ] 모든 작업 체크박스 완료
- [ ] 대시보드에 "설정" 탭 존재
- [ ] 폼에서 모든 핵심 필드 편집 가능
- [ ] 저장 시 CLI 호출 → yaml 반영 → vm.config 갱신
- [ ] 저장 실패 시 alert 표시

**자동 검증:**
- [ ] `go test ./...` 통과 (회귀 없음)
- [ ] `cd helpers && swift build` 통과

**수동 확인 (E2E):**
- [ ] 설정 탭 → workDurationMin 50 → 45로 변경 → Save → 알림 없이 성공 → `cat ~/.config/break-reminder/config.yaml`에서 `work_duration_min: 45` 확인
- [ ] 동일한 시퀀스로 notificationsEnabled 토글 → 데몬 재실행 후 알림 비활성 확인
- [ ] workEndHour을 workStartHour보다 작게 설정 → Save → alert "work schedule must end after it starts" 표시, yaml 변경 안 됨
- [ ] Cancel 버튼 → 폼이 원래 값으로 복귀
- [ ] 설정 탭 → 변경 없이 다른 탭 클릭 → 정상 이동
- [ ] 다크/라이트 테마 양쪽 가독성 OK

**완료 처리:**
1. 사용자에게 완료 보고 — 전체 5개 페이즈 완료
2. 로드맵의 "최종 완료 체크리스트" 수행 권유
3. 사용자 승인 후 (선택) 릴리스 절차 — VERSION bump + tag (CLAUDE.md Release 절차)

---

## 참고 자료

- 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)
- 기존 탭 구조: `TabBarView.swift`, `DashboardAppMain.swift`, `TimerTabView.swift` (탭 컨텐츠 패턴)
- SwiftUI Form: macOS native settings 스타일
- Phase 4의 CLI: `break-reminder config set`
- CLAUDE.md Release 절차 (전체 완료 후 릴리스 시 참조)

## 메모 / 주의

- **macOS 타겟 버전**: `@Observable` 매크로는 macOS 14+. 프로젝트가 13 이하 지원이면 `ObservableObject`/`@Published`/`@StateObject` 사용. `helpers/Package.swift` 또는 `.swift-version`/타겟 설정 확인 후 결정.
- **저장 후 데몬 재시작 필요한 키**: 사실 대부분 키가 데몬이 다음 Tick에서 `config.Load()`를 다시 호출하면 적용되지만, 현재 `daemon.go`는 시작 시 한 번만 로드할 수도 있음. 확인: `cmd/break-reminder/daemon.go` 또는 `check.go`에서 매 호출마다 `config.Load`를 호출하는지. **launchd 60초 사이클이면 매 호출마다 Load 호출하므로 자동 반영됨** — 확인만.
- **work_days Toggle 배치**: 가로 7개는 좁아질 수 있음. 7개 ToggleStyle.button 또는 작은 Checkbox 형태로. 라벨은 "월/화/수/목/금/토/일".
- **에러 메시지 한글화**: `validateSchedule`의 영어 메시지를 그대로 alert에 노출 → 사용자에게 충분히 이해 가능. 굳이 한글화 안 함.
- **민감값 누락 확인**: tts_api_key는 UI에서 절대 노출 안 함. Phase 4의 valid keys에는 있지만 SettingsFormState에 필드 자체를 안 만듦 → 의도된 갭.

---
_이 페이즈가 마지막입니다. 완료 후 로드맵의 "최종 완료 체크리스트" 수행._
