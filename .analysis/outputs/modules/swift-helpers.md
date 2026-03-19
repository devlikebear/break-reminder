# 모듈: Swift 헬퍼 (helpers/)

SPM(Swift Package Manager) 패키지. Go에서 서브프로세스로 실행되는 네이티브 macOS UI 앱.

## 패키지 구조
```
helpers/
├── Package.swift
├── Sources/
│   ├── HelperCore/         # 공유 순수 로직 (테스트 가능)
│   │   ├── ArgsParser.swift
│   │   ├── ConfigParser.swift
│   │   ├── StateParser.swift
│   │   ├── TimeFormatter.swift
│   │   └── ProgressCalc.swift
│   ├── BreakScreenApp/     # 풀스크린 잠금 화면
│   │   └── main.swift
│   └── DashboardApp/       # 네이티브 GUI 대시보드
│       └── main.swift
└── Tests/HelperCoreTests/  # 35개 단위 테스트
```

## HelperCore (공유 라이브러리)

| 파일 | 역할 |
|------|------|
| `ArgsParser` | CLI 인자 파싱 (`--duration`, `--skip-after`, `--work-min`, `--break-min`) + `formatMinutes()` |
| `ConfigParser` | YAML config 간이 파싱 (정규식 없이 line-by-line) |
| `StateParser` | state 파일 파싱 + 직렬화 |
| `TimeFormatter` | 초 → "mm:ss" 포맷 |
| `ProgressCalc` | 작업/휴식 진행률 계산 (실시간 보간 포함) |

## BreakScreenApp

- `NSWindow.Level.screenSaver` 레벨의 borderless 풀스크린
- `KeyWindow` 서브클래스 (`canBecomeKey: true`) — 키보드 이벤트 수신
- 멀티모니터: `NSScreen.screens` 순회, 각 화면에 윈도우 생성
- `collectionBehavior`: `.canJoinAllSpaces`, `.stationary`, `.ignoresCycle`
- localFrame (origin: .zero) 사용으로 보조 모니터에서도 정확한 레이아웃
- 비동기 윈도우 ordering (`DispatchQueue.main.async`)
- Skip 버튼: 최소 2분 후 활성화
- Esc 키로 즉시 종료

## DashboardApp

- 커스텀 `CircularProgressView` (원형 프로그레스 바)
- `StatBarView` (일간 통계 바)
- 1초마다 state 파일 + config 파일 읽기 → 실시간 갱신
- 실시간 보간: `workSeconds + (now - lastCheck)` (60초 체크 간격 보정)
- 키보드: q=종료, r=리셋, b=강제휴식
