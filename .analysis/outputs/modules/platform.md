# 모듈: 플랫폼 추상화

Go build tag를 사용하여 macOS 전용 기능과 스텁을 분리.

## 패턴

```
internal/<package>/
├── <name>.go           # interface 정의
├── <name>_darwin.go    # //go:build darwin — macOS 구현
└── <name>_stub.go      # //go:build !darwin — noop 스텁
```

## idle (유휴 감지)

- **interface**: `Detector.IdleSeconds() int`
- **darwin**: `ioreg -c IOHIDSystem -d 4` 실행 → `HIDIdleTime` 나노초 파싱 → 초 변환
- **stub**: 항상 0 반환

## notify (알림)

- **interface**: `Notifier.Send(title, message, sound) error`
- **darwin**: `osascript -e 'display notification "msg" with title "title" sound name "sound"'`
- **stub**: noop

## tts (음성 합성)

- **interface**: `Speaker.Speak(voice, message) error` + `Available(voice) bool`
- **darwin**: `say -v <voice> "<message>"`. Available은 `say -v ?` 출력에서 검색.
- **stub**: noop

## breakscreen (잠금 화면)

- **`breakscreen.go`**: 오케스트레이션 — mode에 따라 overlay/notification 분기
- **`overlay_darwin.go`**: Swift BreakScreenApp 헬퍼를 `exec.Command`로 실행
- **`overlay_stub.go`**: 알림 폴백
- **`ask_darwin.go`**: `osascript` 다이얼로그로 block/notify 선택
- **`helper.go`**: `FindHelper(name)` — 실행 파일 옆, bin/, ~/.local/bin/, PATH 순으로 검색
