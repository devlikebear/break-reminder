# 모듈: 서비스 인프라

LaunchAgent 관리, 로깅, 스케줄링, 진단 — 시스템 운영 지원 모듈.

---

## launchd (`internal/launchd/`)

macOS LaunchAgent를 통한 백그라운드 스케줄링.

### API

| 함수 | 동작 |
|------|------|
| `Install(binaryPath)` | plist 생성 → `launchctl load` |
| `Uninstall()` | `launchctl unload` → plist 삭제 |
| `Start()` / `Stop()` | load / unload |
| `Status()` | "Not Installed" / "Installed & Running" / "Installed (Not Loaded)" |
| `PlistPath()` | `~/Library/LaunchAgents/com.devlikebear.break-reminder.plist` |

### plist 구조
```xml
<key>ProgramArguments</key>
<array>
    <string>/path/to/break-reminder</string>
    <string>check</string>
</array>
<key>StartInterval</key>
<integer>60</integer>
<key>RunAtLoad</key>
<true/>
```
- 60초 간격으로 `break-reminder check` 실행
- stdout/stderr → `/tmp/break-reminder.{out,err}`

---

## schedule (`internal/schedule/`)

### `IsWorkingTime(cfg, t) bool`
- Config의 `work_days` (ISO: 1=Mon..7=Sun) + `work_start_hour` / `work_end_hour`
- Go의 `time.Weekday()` (0=Sun..6=Sat) → ISO 변환: `0 → 7`
- check 커맨드에서 근무시간 외에는 타이머 비활성

---

## logging (`internal/logging/`)

심플한 파일 로거. 구조화 로깅(zerolog)과 별도로 사용자 가시성 로그.

| 함수 | 동작 |
|------|------|
| `Log(path, msg)` | `[2026-03-19 10:00:00] msg` 형태로 append |
| `Rotate(path, maxLines)` | 초과 시 뒤에서 maxLines만 유지 |
| `Tail(path, n)` | 마지막 n줄 반환 (대시보드용) |

- **파일 경로**: `~/.break-reminder.log`
- 대시보드에서 최근 5줄을 `Tail()`로 읽어 표시

---

## doctor (`internal/doctor/`)

시스템 진단 — 모든 구성 요소의 정상 동작 여부를 한 번에 확인.

### Check 항목

| 체크 | 방법 |
|------|------|
| Voice | `tts.Available(voice)` |
| TTS | `tts.Speak()` 실행 |
| Notification | `notify.Send()` 실행 |
| Idle detection | `idle.IdleSeconds()` 호출 |
| State file | `os.Stat()` 존재 확인 |
| Log file | `os.Stat()` + 크기 확인 |
| LaunchAgent | `launchd.Status()` |
| Working hours | `schedule.IsWorkingTime()` |
| Config file | `os.Stat()` 존재 확인 |

### Report 구조
```go
type Check struct {
    Name   string
    Status string // "ok", "warn", "fail", "info"
    Detail string
}

type Report struct {
    Checks []Check
}
```

- `Run(cfg) Report` — 모든 체크를 순서대로 실행
- `FailCount()` — 실패 수 반환 (exit code에 사용)
