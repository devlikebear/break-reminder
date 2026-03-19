# 용어집

| 용어 | 설명 |
|------|------|
| **Tick** | 60초마다 실행되는 타이머 체크 1회. `timer.Tick()` 함수. |
| **Work mode** | 작업 모드. 기본 50분. 유휴가 아닌 시간이 누적됨. |
| **Break mode** | 휴식 모드. 기본 10분. 휴식 시작 시각 기준으로 경과 계산. |
| **Natural break** | 유휴 시간이 `natural_break_sec`(기본 5분)을 초과하면 자동으로 작업 타이머 리셋. |
| **Idle threshold** | 유휴 판단 기준 (기본 120초). 이 이상이면 작업 시간 누적 중지. |
| **Break screen mode** | "ask"(첫 번째에 선택), "block"(풀스크린 잠금), "notify"(알림만). |
| **LaunchAgent** | macOS의 사용자 레벨 데몬. plist 파일로 등록, `launchctl`로 관리. |
| **HelperCore** | Swift 공유 라이브러리. UI 앱에서 사용하는 순수 로직 (파싱, 계산). |
| **DayEndSummary** | 일간 리셋 시 이전 날의 작업/휴식 통계. 히스토리 파일에 저장. |
| **Gap detection** | 체크 간격이 비정상적으로 길면 (슬립 등) 해당 시간을 작업에 포함하지 않음. |
| **SPM** | Swift Package Manager. helpers/ 디렉토리의 빌드 시스템. |
| **Bubbletea** | Go TUI 프레임워크. Elm 아키텍처 (Model-Update-View). |
| **Lipgloss** | Bubbletea용 스타일링 라이브러리. |
