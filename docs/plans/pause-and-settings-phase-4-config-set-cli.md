# Phase 4: config set CLI + HelperCore 확장 — 작업지시서

_작성일: 2026-05-12_
_속한 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)_
_예상 소요: 2시간_

## 페이즈 목표

CLI 한 줄로 모든 설정 키를 변경할 수 있다 — `break-reminder config set work_duration_min=45`. 검증(`validateSchedule`)과 원자 쓰기를 거치므로 데몬과의 충돌 없음. Swift HelperCore의 `AppConfig`/`parseConfig`도 핵심 8-10개 필드를 모두 다룰 수 있도록 확장 (저장은 Phase 5에서 CLI에 위임).

## 전제 조건

- [ ] Phase 3 완료 및 사용자 승인 (또는 Phase 2 완료 후 Phase 3과 병렬 진행 시 Phase 2 승인)
- [ ] `internal/config/load.go`의 `ApplyYAMLChanges`/`Save` 동작 이해

## 포함 기능

1. `break-reminder config set <key>=<value> [<key>=<value> ...]` 명령
2. `break-reminder config get <key>` 명령 (선택, 편의용)
3. `internal/config.ApplyYAMLChanges` 재활용 — 키 화이트리스트와 검증 그대로
4. `AppConfig` Swift struct에 핵심 필드 모두 노출
5. `parseConfig`에서 work_days 같은 배열 처리

## 이 페이즈에서 하지 않는 것

- GUI Settings 탭 → Phase 5
- 잘못된 키/값에 대한 에러 메시지 정교화 (현재 `validateSchedule` 메시지 그대로 사용)
- `config reset` 같은 기본값 복원 명령 → Phase 5에서 GUI 버튼으로 처리 또는 Out of Scope

## 작업 체크리스트

### 작업 그룹 A: `config set` 서브명령

- [ ] **T4.A.1** — `config set` 명령 추가
  - 파일: `cmd/break-reminder/config_cmd.go`
  - 내용:
    - `newConfigCmd()` 내부에 추가:
      ```go
      setCmd := &cobra.Command{
          Use:   "set <key=value> [<key=value> ...]",
          Short: "Set one or more configuration values",
          Args:  cobra.MinimumNArgs(1),
          RunE: func(cmd *cobra.Command, args []string) error {
              // 1. 인자 파싱: "key=value" → map[string]any
              changes := map[string]any{}
              for _, arg := range args {
                  k, v, ok := strings.Cut(arg, "=")
                  if !ok || k == "" {
                      return fmt.Errorf("invalid argument %q (expected key=value)", arg)
                  }
                  changes[strings.TrimSpace(k)] = parseConfigValue(strings.TrimSpace(v))
              }
              // 2. YAML로 직렬화
              data, err := yaml.Marshal(changes)
              if err != nil {
                  return err
              }
              // 3. ApplyYAMLChanges 호출 (검증 포함)
              updated, err := config.ApplyYAMLChanges(cfg, data)
              if err != nil {
                  return fmt.Errorf("invalid config change: %w", err)
              }
              // 4. 저장
              if err := config.Save(updated); err != nil {
                  return fmt.Errorf("save config: %w", err)
              }
              fmt.Fprintf(cmd.OutOrStdout(), "Configuration updated (%d keys).\n", len(changes))
              return nil
          },
      }
      cmd.AddCommand(setCmd)
      ```
    - `parseConfigValue(v string) any` 헬퍼:
      - `v == "true" || v == "false"` → bool
      - `strconv.Atoi`로 성공하면 int
      - `[` 로 시작하면 yaml.Unmarshal로 list 파싱 (work_days용)
      - 그 외 → string 그대로
    - allowInvalidConfig(setCmd) 호출 여부: set 명령은 cfg 유효 상태가 필요하므로 호출 X (기본값 유효 cfg 요구)
  - 참조: 기존 `newConfigCmd`의 다른 서브명령 (`show`, `edit`, `path`) 패턴
  - 검증: `break-reminder config set work_duration_min=45` 후 `break-reminder config show`에 반영

- [ ] **T4.A.2** — `config get` 명령 (선택)
  - 파일: `cmd/break-reminder/config_cmd.go`
  - 내용:
    - 단일 키 조회: `break-reminder config get work_duration_min`
    - reflect 사용하거나 키별 switch 단순 매핑
    - 우선순위 낮음 — 시간 부족하면 스킵. 단, set이 잘 됐는지 확인하려면 show로 충분하니 정말 선택적.
  - 검증: (구현 시) 수동

### 작업 그룹 B: 단위 테스트

- [ ] **T4.B.1** — `config set` 명령 테스트
  - 파일: `cmd/break-reminder/config_set_test.go` (신규)
  - 내용 (table-driven):
    - 케이스 A: `set work_duration_min=45` → 성공, 파일에 반영
    - 케이스 B: `set work_start_hour=25` → `validateSchedule` 에러로 거부
    - 케이스 C: `set unknown_key=foo` → ApplyYAMLChanges가 `unknown config key` 에러
    - 케이스 D: `set work_duration_min=45 break_duration_min=15` → 두 값 모두 반영
    - 케이스 E: `set notifications_enabled=false` → bool 처리 정상
  - 테스트 헬퍼: `t.TempDir()`로 임시 HOME 만들고 환경변수 override (기존 테스트가 이 패턴을 쓰는지 확인 — `config_test.go` 참조)
  - 검증: `go test ./cmd/break-reminder/... -run TestConfigSet -v` 통과

### 작업 그룹 C: HelperCore Swift 확장

- [ ] **T4.C.1** — `AppConfig`에 핵심 필드 추가
  - 파일: `helpers/Sources/HelperCore/ConfigParser.swift`
  - 내용:
    - 현재 5개 필드 → 핵심 10개 필드로 확장:
      ```swift
      public struct AppConfig: Equatable {
          public var workDurationMin: Int = 50
          public var breakDurationMin: Int = 10
          public var idleThresholdSec: Int = 120
          public var naturalBreakSec: Int = 300
          public var checkIntervalSec: Int = 60
          public var workDays: [Int] = [1,2,3,4,5]
          public var workStartHour: Int = 9
          public var workStartMinute: Int = 0
          public var workEndHour: Int = 18
          public var workEndMinute: Int = 0
          public var notificationsEnabled: Bool = true
          public var ttsEnabled: Bool = true
          public var breakActivitiesEnabled: Bool = true
          public var breakScreenMode: String = "ask"  // "ask|block|notify"
          public var theme: String = "auto"
          public init() {}
      }
      ```
  - 검증: `swift build` 통과

- [ ] **T4.C.2** — `parseConfig` 확장 (배열/bool 처리)
  - 파일: `helpers/Sources/HelperCore/ConfigParser.swift`
  - 내용:
    - 기존 switch 확장 — 모든 신규 필드에 case 추가
    - bool: `val == "true"` 처리
    - work_days 배열: `[1, 2, 3, 4, 5]` 또는 `[1,2,3,4,5]` 형태. 정규식이나 단순 trim + split으로 파싱:
      ```swift
      case "work_days":
          let stripped = val.trimmingCharacters(in: CharacterSet(charactersIn: "[]"))
          c.workDays = stripped.split(separator: ",").compactMap { Int($0.trimmingCharacters(in: .whitespaces)) }
      ```
    - 검증: 단위 테스트(T4.C.3)
  - 참조: 기존 5개 필드 처리 그대로
  - 검증: `swift build` 통과

- [ ] **T4.C.3** — HelperCore 테스트 (있으면)
  - 파일: `helpers/Tests/HelperCoreTests/ConfigParserTests.swift` (존재 여부 확인)
  - 내용:
    - `default.yaml` 내용을 그대로 파싱했을 때 기본값과 일치
    - `work_days: [1, 2, 3]` 파싱 정확
    - `notifications_enabled: false` 파싱 정확
  - 검증: (테스트 디렉토리 존재 시) `swift test`

---

## ✅ Phase 4 Checkpoint

**구현 확인:**
- [ ] 모든 작업 체크박스 완료
- [ ] `break-reminder config set k=v`가 yaml에 정확히 반영
- [ ] 잘못된 키/값은 검증으로 거부됨
- [ ] Swift `AppConfig`가 핵심 10개 필드 모두 읽을 수 있음

**자동 검증:**
- [ ] `go test ./...` 통과
- [ ] `cd helpers && swift build` 통과

**수동 확인:**
- [ ] `break-reminder config set work_duration_min=45` → `cat ~/.config/break-reminder/config.yaml | grep work_duration_min` 결과 `work_duration_min: 45`
- [ ] `break-reminder config set work_duration_min=99 work_start_hour=25` → 에러 반환, 파일 변경 안 됨 (atomic)
- [ ] `break-reminder config set notifications_enabled=false` → 파일에 반영, 데몬 재시작 후 알림 안 옴
- [ ] GUI 대시보드 켜서 새로 추가된 필드들이 정상 표시되는지 (예: `naturalBreakSec` 같은 값이 InsightsTabView나 StatsTabView에 노출되는지 — 노출 안 해도 됨, parsing이 안 깨지는지만)

**완료 처리:**
1. 사용자에게 완료 보고
2. 사용자 승인 후 Phase 5로 이동.
3. 실패 시: 원인 분석 → 수정 → 재검증.

---

## 참고 자료

- 로드맵: [`pause-and-settings-roadmap.md`](./pause-and-settings-roadmap.md)
- 기존 config save/validate: `internal/config/load.go:80-149`
- 기존 config 명령 구조: `cmd/break-reminder/config_cmd.go`
- yaml 라이브러리: `gopkg.in/yaml.v3` (이미 의존성에 있음)
- CLAUDE.md "Key Review Checklist" — "config validation: `merge()` 에서 zero-value와 null 값 구분"

## 메모 / 주의

- **`validateSchedule`이 호출되는 시점**: `ApplyYAMLChanges` 내부에서 한 번, `config.Save` 내부에서 한 번. 중복이지만 안전성 ↑. 두 검증이 같은 결과를 내므로 그대로 둠.
- **bool 처리 함정**: `merge()` 함수는 raw map에 키가 있을 때만 dst 갱신. `parseConfigValue`가 bool을 정확히 `true`/`false`(Go bool)로 반환해야 함. 문자열 `"false"`로 들어가면 yaml.Marshal이 `notifications_enabled: "false"` (string)로 직렬화하고 unmarshal 시 `bool(true)`로 인식될 위험. 반드시 Go bool 타입으로 넣을 것.
- **work_days 배열 처리**: `parseConfigValue("[1,2,3]")`는 `[]any{1,2,3}`을 반환해야 함. yaml.Unmarshal로 우회하면 안전 — flow style 배열 지원.
- **민감 키 제외**: tts_api_key는 valid keys에는 있지만 CLI set으로 노출되어도 무방 (사용자 본인이 입력). Phase 5 GUI에서는 의도적으로 빼므로 일관성 유지.

---
_다음 페이즈: Phase 5 — Settings UI 탭 → [`pause-and-settings-phase-5-settings-ui.md`](./pause-and-settings-phase-5-settings-ui.md)_
