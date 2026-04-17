# DashboardApp v2 — Design Spec

> Date: 2026-04-17
> Status: Approved
> Target: Swift DashboardApp (native macOS GUI)

## Overview

Break Reminder의 Swift DashboardApp을 SwiftUI로 전면 마이그레이션하고, AI Summary, 데이터 시각화, 비주얼 강화 기능을 추가하여 사용 재미를 향상시킨다.

## Implementation Strategy

단계적 구현 (Phase별 독립 릴리즈 가능):

1. **Phase 1:** SwiftUI 마이그레이션 — 현재 AppKit 기능을 SwiftUI로 1:1 포팅
2. **Phase 2:** 탭 레이아웃 + 데이터 시각화 (Swift Charts)
3. **Phase 3:** AI Summary 연동 (일일 리포트, 패턴 인사이트, 코칭)
4. **Phase 4:** 비주얼 강화 (마스코트, 애니메이션, 테마)

## Architecture

### 전체 레이아웃

360×600 고정 크기 플로팅 윈도우. 상단 고정 영역 + 하단 탭 전환 구조.

```
┌──────────────────────────────────┐
│  상단 고정 영역                    │
│  ┌─ 상태 dot + 라벨 ──────────┐  │
│  │  ● WORKING       18/25 min │  │
│  └────────────────────────────┘  │
│         ┌──────────┐             │
│         │  07:00   │  타이머 링   │
│         │until break│            │
│         └──────────┘             │
│     🐹 집중 잘 하고 있어요!        │
├──────────────────────────────────┤
│  [타이머]  [통계]  [인사이트]  탭 바 │
├──────────────────────────────────┤
│                                  │
│  탭 콘텐츠 영역                    │
│                                  │
└──────────────────────────────────┘
```

### 상단 고정 영역 (모든 탭에서 보임)

- 상태 표시: dot(녹색/파랑/노랑) + 라벨(WORKING/ON BREAK/PAUSED) + 모드 정보
- 원형 프로그레스 링: 현재 세션 진행률, 남은 시간 표시
- 마스코트 + 코칭 메시지: 이모지 + 말풍선

### 탭 구조

3개 탭: 타이머 / 통계 / 인사이트

## Phase 1: SwiftUI 마이그레이션

현재 AppKit `DashboardApp/main.swift`의 기능을 SwiftUI로 1:1 포팅.

### 마이그레이션 대상

| AppKit (현재) | SwiftUI (신규) |
|---|---|
| `CircularProgressView` (NSView) | SwiftUI `Canvas` 또는 `Shape` |
| `StatBarView` (NSView) | `ProgressView` 또는 커스텀 `Shape` |
| `DashboardApp` (NSApplicationDelegate) | SwiftUI `App` + `@main` |
| 수동 프레임 레이아웃 | SwiftUI 선언적 레이아웃 |
| `NSTextField` 라벨들 | SwiftUI `Text` |
| `NSButton` | SwiftUI `Button` |
| `Timer.scheduledTimer` 1초 갱신 | SwiftUI `TimelineView` 또는 `.onReceive(Timer)` |

### 데이터 흐름

현재 패턴 유지: 1초마다 `~/.break-reminder-state` 파일을 읽어서 UI 갱신.

```
~/.break-reminder-state (Go가 작성) → Swift가 1초마다 읽기 → @State 업데이트 → UI 갱신
~/.config/break-reminder/config.yaml → Swift가 1초마다 읽기 → 설정 반영
```

### 윈도우 속성 유지

- 크기: 360×600 (520에서 확장)
- 위치: 화면 우상단
- 스타일: 투명 타이틀바, floating level, 배경 드래그 가능
- 키보드: q(종료), r(리셋), b(강제 휴식)

### 기존 HelperCore 의존성

`StateParser`, `ConfigParser`, `ProgressCalc`, `TimeFormatter` 등 HelperCore 라이브러리는 그대로 활용.

## Phase 2: 탭 레이아웃 + 데이터 시각화

### 타이머 탭 (기본 탭)

현재 DashboardApp의 하단 영역을 이동:
- 일일 통계: 작업 시간, 휴식 시간, 비율 바
- 시스템 정보: launchd 상태, idle 시간
- 액션 버튼: Reset, Force Break

### 통계 탭

Swift Charts 프레임워크 사용 (macOS 13+).

**구성 요소:**

1. **기간 선택 세그먼트:** 주간 / 월간 / 전체
2. **작업/휴식 스택 바 차트:** `BarMark` — 요일별 작업(녹색) + 휴식(파랑) 스택
3. **시간대별 집중도 히트맵:** SwiftUI `Grid` + `RoundedRectangle` — GitHub 스타일. 행=요일, 열=시간(9~18시)
4. **요약 카드:** 3열 그리드 — 주간 작업 총량, 휴식 총량, 작업 비율

**데이터 소스:**

- 기존 `~/.break-reminder-history.json` (`DailySummary`: date, work_min, break_min, sessions, activities)
- 히트맵용 시간대별 데이터: `DailySummary`에 `hourly_work` 필드 추가 (배열: 24개 분 값)
  - Go `internal/ai/history.go`의 `DailySummary` 구조체 확장
  - Go 타이머 tick에서 시간대별 누적 로직 추가

**Swift Charts 최소 요구사항:** macOS 13 (Ventura) 이상.

### 히스토리 포맷 확장

```json
{
  "date": "2026-04-17",
  "work_min": 280,
  "break_min": 60,
  "sessions": 7,
  "activities": 3,
  "hourly_work": [0,0,0,0,0,0,0,0,0,45,55,50,10,40,50,35,20,0,0,0,0,0,0,0]
}
```

기존 필드와 하위 호환 유지. `hourly_work`가 없는 기존 데이터는 히트맵에서 빈칸 처리.

## Phase 3: AI Summary 연동

### AI 호출 흐름

```
Go 메인 프로세스 (launchd tick 또는 CLI 명령)
  ↓
ai.Client.Query(prompt) — claude/codex CLI 호출
  ↓
결과 파싱 → ~/.break-reminder-insights.json 저장
  ↓
Swift DashboardApp — 파일 읽기 → 인사이트 탭 렌더링
```

### 인사이트 파일 포맷

```json
{
  "generated_at": "2026-04-17T17:30:00+09:00",
  "daily_report": "오늘 4시간 20분 작업하고 50분 휴식했어요. 오전에 집중력이 높았고...",
  "patterns": [
    {
      "type": "warning",
      "title": "오후 슬럼프 패턴 감지",
      "description": "최근 5일 중 4일, 오후 2시~4시에 평균 작업 시간이 35% 줄었어요.",
      "suggestion": "이 시간대에 짧은 산책을 추가하면 효과적일 수 있어요."
    },
    {
      "type": "positive",
      "title": "휴식 습관 개선 중",
      "description": "지난주 대비 휴식 건너뛰기가 40% 줄었어요.",
      "suggestion": "꾸준히 유지하면 집중력 향상에 도움이 됩니다."
    },
    {
      "type": "info",
      "title": "최적 작업 시간대",
      "description": "오전 10시~12시가 가장 집중도가 높은 골든 타임이에요.",
      "suggestion": "중요한 작업은 이 시간에 배치하면 좋겠어요."
    }
  ]
}
```

### 인사이트 탭 UI

1. **오늘의 리포트:** `daily_report` 텍스트를 카드형으로 표시. 녹색 좌측 보더.
2. **패턴 인사이트:** `patterns` 배열을 카드 리스트로 표시. type별 dot 색상 (warning=노랑, positive=녹색, info=파랑).
3. **액션 버튼:**
   - "AI 분석 새로고침" — Swift에서 `Process()`로 `break-reminder insights --refresh` CLI 실행 (기존 `findHelper()` 패턴 활용). 실행 중 스피너 표시, 완료 후 파일 재로드.
   - "리포트 복사" — `NSPasteboard`로 일일 리포트 텍스트를 클립보드에 복사

### AI 프롬프트 설계

Go 쪽에서 히스토리 데이터를 수집하여 프롬프트로 구성:

```
다음은 사용자의 최근 7일 작업/휴식 기록입니다:
[히스토리 JSON]

1. 오늘의 요약을 2-3문장으로 작성하세요.
2. 눈에 띄는 패턴 2-3가지를 분석하세요 (각각 type, title, description, suggestion).
3. JSON 형식으로 응답하세요.
```

### AI CLI 미설치 시

인사이트 탭에 안내 메시지 표시:
> "AI CLI(claude 또는 codex)를 설치하면 일일 리포트와 패턴 분석을 볼 수 있습니다."

### 실시간 코칭 (상단 고정 영역)

마스코트 말풍선으로 표시. 규칙 기반 (AI 호출 X):

| 조건 | 메시지 예시 |
|---|---|
| 작업 시작 | "집중 모드! 화이팅 💪" |
| 남은 시간 5분 | "곧 휴식 시간이에요~ ☕" |
| 연속 2시간 작업 | "쉬어가는 게 어때요? 🙏" |
| 휴식 중 | "푹 쉬고 와요~" |
| 일일 목표 달성 | "오늘도 잘 해냈어요! 🏆" |
| 아침 시작 | "좋은 아침! 오늘도 파이팅 🌅" |

AI 인사이트에서 핵심 메시지를 추출하여 코칭에 반영하는 것도 가능 (예: "오늘 오후 슬럼프 주의!").

### AI 분석 트리거 타이밍

- **자동 트리거:** 하루 1회, 근무 종료 시간(config의 working_hours 기준) 즈음에 실행
- **수동 트리거:** 인사이트 탭의 "새로고침" 버튼 또는 CLI `break-reminder insights --refresh`
- AI CLI 호출은 비동기로 처리하여 메인 타이머 루프를 블로킹하지 않음

## Phase 4: 비주얼 강화

### 마스코트 시스템

시스템 이모지 기반 (v1). 상태와 조건에 따라 이모지 + 메시지 조합.

**상태 매핑:**

| 상태 | 이모지 | 말풍선 톤 |
|---|---|---|
| 작업 중 (정상) | 🐹 | 격려, 응원 |
| 휴식 중 | 😴 | 편안함, 휴식 독려 |
| 장시간 연속 작업 | 😰 | 걱정, 휴식 권유 |
| 일일 목표 달성 | 🎉 | 축하, 칭찬 |
| 일시정지 | 😶 | 대기 |
| 근무 시간 외 | 🌙 | 퇴근 축하 |

**구현:** `MascotEngine` 구조체 — 현재 상태/조건을 받아 (emoji, message) 튜플 반환.

향후 v2에서 커스텀 일러스트(Lottie 또는 에셋 기반)로 업그레이드 가능.

### 애니메이션 효과

| 효과 | 적용 위치 | SwiftUI 구현 |
|---|---|---|
| 프로그레스 링 글로우 | 타이머 링 끝점 | `.shadow(color:radius:)` + `Animation.easeInOut.repeatForever` |
| 상태 전환 트랜지션 | work↔break | `withAnimation(.easeInOut(duration: 0.5))` 색상 모프 |
| 목표 달성 컨페티 | 전체 화면 오버레이 | `Canvas` + `TimelineView` 파티클 시스템 |
| 탭 전환 | 탭 콘텐츠 | `.transition(.slide)` + `.animation(.easeInOut)` |
| 마스코트 바운스 | 마스코트 이모지 | `.scaleEffect` + `Animation.spring` on state change |

### 테마 시스템

3가지 모드: 다크 (기본) / 라이트 / 자동 (시스템 연동)

**구현:**

```swift
// ThemeManager — 컬러 토큰 추상화
class ThemeManager: ObservableObject {
    @Published var mode: ThemeMode = .auto  // .dark, .light, .auto

    var background: Color { ... }
    var surface: Color { ... }
    var textPrimary: Color { ... }
    var textSecondary: Color { ... }
    var accent: Color { ... }        // work color
    var accentBreak: Color { ... }   // break color
    var warning: Color { ... }
}
```

- `@Environment(\.colorScheme)`으로 시스템 테마 감지
- `config.yaml`에 `theme: auto | dark | light` 옵션 추가
- `ThemeManager`를 `@EnvironmentObject`로 전체 뷰 트리에 주입

**컬러 팔레트:**

| 토큰 | 다크 | 라이트 |
|---|---|---|
| background | #1a1a1e | #f5f5f7 |
| surface | #252528 | #ffffff |
| textPrimary | #e5e5e5 | #1a1a1e |
| textSecondary | #888888 | #666666 |
| accent (work) | #4dcc80 | #34a853 |
| accentBreak | #66b3ff | #4285f4 |
| warning | #ffcc66 | #f9ab00 |

## 데이터 흐름 요약

```
Go 메인 프로세스 (launchd, 60초 체크)
  ├─ ~/.break-reminder-state        → 타이머 상태 (1초 갱신 by Go)
  ├─ ~/.config/break-reminder/config.yaml → 사용자 설정
  ├─ ~/.break-reminder-history.json → 일간 히스토리 (hourly_work 확장)
  └─ ~/.break-reminder-insights.json → AI 인사이트 (하루 1회 갱신)

Swift DashboardApp (1초 폴링)
  ├─ state 파일 → 상단 고정 영역 + 타이머 탭
  ├─ config 파일 → 테마, 설정값
  ├─ history 파일 → 통계 탭 (차트, 히트맵)
  └─ insights 파일 → 인사이트 탭 + 코칭 메시지
```

## 호환성 / 제약 사항

- **최소 macOS 버전:** macOS 13 (Ventura) — Swift Charts 요구사항
- **AI CLI 의존성:** 선택적. 미설치 시 인사이트 탭은 안내 메시지만 표시
- **기존 TUI 대시보드:** 영향 없음. Go TUI(`internal/dashboard`)는 별도로 유지
- **MenuBarApp:** 영향 없음. 독립 바이너리
- **HelperCore:** `StateParser`, `ConfigParser`, `ProgressCalc` 등 기존 라이브러리 재사용
- **히스토리 포맷:** `hourly_work` 필드 추가는 하위 호환. 기존 데이터는 히트맵에서 빈칸 처리

## 테스트 전략

- **HelperCore:** 기존 단위 테스트 유지 + 신규 파서/모델 테스트 추가
- **MascotEngine:** 상태→이모지/메시지 매핑 단위 테스트
- **ThemeManager:** 모드별 컬러 반환 테스트
- **인사이트 파싱:** JSON 파싱 + 누락 필드 처리 테스트
- **UI 수동 테스트:** 각 Phase 완료 시 DashboardApp 실행하여 시각적 확인
