# Issue Candidates

> analyze 모드에서 발견된 리팩터링 후보. `refactor-guide` 모드에서 상세 분석용.

### TIDY-001: key=value 상태 파일 형식

- Module: `internal/state`
- Type: `TIDY`
- Evidence: bash 스크립트 호환을 위해 key=value 형식을 사용하나, Go 전용으로 전환 완료. JSON/YAML 등 구조화 형식이 더 적합.
- Suggested action: 상태 파일을 JSON 형식으로 마이그레이션 (하위 호환 고려).

### TIDY-002: HelperCore에 중복 파서 구현

- Module: `helpers/Sources/HelperCore`
- Type: `DUP`
- Evidence: ConfigParser, StateParser가 Go 쪽 config/state 패키지와 동일 로직을 Swift로 재구현. 프로세스간 통신이 CLI 인자로만 이루어져 파싱 로직 중복 발생.
- Suggested action: Go에서 JSON으로 데이터 전달하여 Swift 쪽 파서 단순화 검토.

### TIDY-003: doctor 패키지 과다 의존성

- Module: `internal/doctor`
- Type: `TIDY`
- Evidence: doctor.go가 config, idle, launchd, logging, notify, schedule, state, tts 등 8개 패키지에 의존. 진단 특성상 불가피하나, interface 기반 주입으로 결합도 완화 가능.
- Suggested action: 각 체크를 인터페이스로 추상화하여 doctor가 직접 의존하지 않도록 리팩터링.

### SEC-001: AI CLI 서브프로세스 프롬프트 인젝션

- Module: `internal/ai`
- Type: `SEC`
- Evidence: `ai configure` 커맨드가 사용자 자연어 입력을 직접 프롬프트에 포함하여 외부 CLI로 전달. 프롬프트 인젝션 가능성.
- Suggested action: 사용자 입력에 대한 sanitization 또는 구조화된 프롬프트 템플릿 적용.

### TIDY-004: break-reminder.sh 레거시 스크립트

- Module: `break-reminder.sh`
- Type: `TIDY`
- Evidence: Go 재작성 완료 후 bash 스크립트가 참조용으로만 남아 있음. 유지보수 대상에서 혼란 유발 가능.
- Suggested action: 레거시 스크립트를 docs/ 또는 archive/로 이동하거나, README에 deprecated 명시.
