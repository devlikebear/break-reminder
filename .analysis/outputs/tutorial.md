# 튜토리얼: Break Reminder 시작하기

## 1. 설치

### Homebrew (권장)
```bash
brew install devlikebear/tap/break-reminder
```

### 소스에서 빌드
```bash
git clone https://github.com/devlikebear/break-reminder.git
cd break-reminder
make build        # bin/ 디렉토리에 바이너리 생성
make install      # ~/.local/bin/에 설치 + LaunchAgent 등록
```

## 2. 시스템 진단
```bash
break-reminder doctor
```
TTS, 알림, 유휴 감지, LaunchAgent 등 모든 구성 요소를 확인합니다.

## 3. 서비스 등록
```bash
break-reminder service install    # 60초마다 자동 체크
break-reminder service status     # 상태 확인
```

## 4. 상태 확인
```bash
break-reminder status
```
출력 예시:
```
🐹 Break Reminder Status
========================
System: Installed & Running
State:  Active (Within working hours)
------------------------
Mode: work
Session Work: 25min / 50min
Daily Stats: Work 2h 5m / Break 30m
Current idle: 3sec
```

## 5. 대시보드
```bash
break-reminder dashboard          # TUI 대시보드
break-reminder dashboard --gui    # 네이티브 macOS GUI
```

## 6. 설정 변경
```bash
break-reminder config edit        # $EDITOR로 직접 편집
break-reminder config show        # 현재 설정 확인
```

주요 설정:
```yaml
work_duration_min: 50        # 작업 시간 (분)
break_duration_min: 10       # 휴식 시간 (분)
break_screen_mode: "block"   # 풀스크린 잠금
tts_enabled: true            # 음성 안내
ai_enabled: true             # AI 기능 활성화
```

## 7. AI 기능 (선택)
```bash
# config에서 ai_enabled: true 설정 후
break-reminder ai summary          # 오늘의 생산성 리포트
break-reminder ai summary --weekly  # 주간 리포트
break-reminder ai suggest           # 최적 타이밍 제안
break-reminder ai configure "25분 작업, 5분 휴식으로 바꿔줘"
```

## 8. 가이드 휴식
```bash
break-reminder break eye       # 20-20-20 눈 운동
break-reminder break stretch   # 스트레칭 가이드
break-reminder break breathe   # 박스 호흡법
break-reminder break walk      # 산책 타이머
```
