# 🐹 Break Reminder

**A smart break reminder for macOS that monitors your activity and enforces healthy work habits.**

Work 50 minutes → Rest 10 minutes → Repeat!

![macOS](https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white)
![Swift](https://img.shields.io/badge/Swift-5.9-F05138?style=flat&logo=swift&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## ✨ Features

- **🖱️ Activity Detection** — Monitors keyboard/mouse idle time via `ioreg`
- **⏰ Smart Timer** — 50 min work / 10 min break cycle (configurable)
- **🖥️ Fullscreen Break Screen** — Swift AppKit overlay with multi-monitor support
- **📊 TUI Dashboard** — Real-time progress bars and daily statistics (Bubbletea)
- **🖼️ Native GUI Dashboard** — macOS native window with circular progress bar
- **🧘 Guided Break Activities** — Eye exercise, stretching, box breathing, walk timer
- **🤖 AI Integration** — Productivity analysis via Claude/Codex CLI
- **🗓️ Smart Scheduling** — Only active during configured working hours/days
- **📈 Daily Stats** — Track work/break time with gap detection for accurate tracking
- **🔔 Notifications** — Visual + voice alerts (`say`, KittenTTS, or Supertonic)
- **🚀 Auto-start** — LaunchAgent service with 60-second check interval
- **🏥 Diagnostics** — `doctor` command to verify all components

## 📦 Installation

### Homebrew (Recommended)

```bash
brew install devlikebear/tap/break-reminder
```

### From Source

```bash
git clone https://github.com/devlikebear/break-reminder.git
cd break-reminder
make build      # Build Go binary + Swift helpers
make install    # Install to ~/.local/bin/ + register LaunchAgent
```

## 🚀 Usage

### Commands

```bash
# Core
break-reminder check              # Single timer tick (used by launchd)
break-reminder daemon             # Foreground loop
break-reminder status             # Current state overview
break-reminder reset              # Reset timer

# Dashboard
break-reminder dashboard          # TUI dashboard
break-reminder dashboard --gui    # Native macOS GUI

# Service Management
break-reminder service install    # Register LaunchAgent
break-reminder service uninstall  # Remove LaunchAgent
break-reminder service start      # Start service
break-reminder service stop       # Stop service
break-reminder service status     # Check service status

# Guided Break Activities
break-reminder break eye          # 20-20-20 eye exercise (2 min)
break-reminder break stretch      # Stretching guide (5 min)
break-reminder break breathe      # Box breathing (4 min)
break-reminder break walk         # Walk timer (5 min)

# AI (requires claude or codex CLI)
break-reminder ai summary         # Daily productivity report
break-reminder ai summary --weekly # Weekly report
break-reminder ai suggest         # Optimal timing suggestions
break-reminder ai configure "25분 작업, 5분 휴식" # Natural language config

# Configuration
break-reminder config show        # Show current config
break-reminder config edit        # Open in $EDITOR

# Optional TTS backends
break-reminder tts install kittentts   # Install KittenTTS into a managed venv
break-reminder tts install supertonic  # Install Supertonic into a managed venv
break-reminder tts test "안녕하세요"  # Speak a phrase with the current TTS config
break-reminder tts uninstall kittentts   # Remove the managed KittenTTS venv
break-reminder tts uninstall supertonic  # Remove the managed Supertonic venv
break-reminder config path        # Show config file path

# Diagnostics
break-reminder doctor             # Check all components
break-reminder version            # Show version
```

### Dashboard

```
🐹 Break Reminder Dashboard (q:quit r:reset b:break)
══════════════════════════════════════════════════
System: Installed & Running
Status: WORKING
Idle: 3s / Limit: 120s

Session Work: [████████████████░░░░░░░░░░░░░░] 53% (26 / 50 min)

Daily Statistics:
  Work: 2h 5m
  Rest: 30m
  Ratio: [████████████████████░░░░] 80%

Recent Logs:
──────────────────────────────────────────────────
  [2026-03-19 14:30:00] work mode, session 26min
──────────────────────────────────────────────────
```

### Status Output

```
🐹 Break Reminder Status
========================
System: Installed & Running
State:  Active (Within working hours)
------------------------
Mode: work
Session Work: 32min / 50min
Daily Stats: Work 2h 5m / Break 30m
Current idle: 3sec
```

## ⚙️ Configuration

Edit `~/.config/break-reminder/config.yaml`:

```yaml
# Timer Settings
work_duration_min: 50          # Work duration (default: 50 min)
break_duration_min: 10         # Break duration (default: 10 min)
idle_threshold_sec: 120        # Idle threshold (default: 2 min)
natural_break_sec: 300         # Auto-reset threshold (default: 5 min)

# Smart Scheduling
work_days: [1, 2, 3, 4, 5]    # ISO weekdays (1=Mon, 7=Sun)
work_start_hour: 9
work_start_minute: 0          # Optional minute precision for the work start boundary
work_end_hour: 18
work_end_minute: 0            # Optional minute precision for the work end boundary

# Break Screen
break_screen_mode: "ask"       # "ask" (choose once), "block" (fullscreen), "notify" (notification only)
break_activities_enabled: true  # Show guided activity menu on break

# Voice & Notifications
voice: "Yuna"                  # Voice name for the selected TTS engine
tts_engine: "say"              # "say", "kittentts", or "supertonic"
tts_model: "KittenML/kitten-tts-nano-0.8"  # Used by KittenTTS; Supertonic currently uses a fixed model bundle
tts_python_cmd: "python3"      # Python with the selected optional TTS package installed
tts_enabled: true
notifications_enabled: true

# AI
ai_enabled: true               # Enable AI features
ai_cli: "claude"               # "claude" or "codex"
```

Or use natural language:

```bash
break-reminder ai configure "25분 작업, 5분 휴식으로 바꿔줘"
```

## 📁 Project Structure

```
cmd/break-reminder/           # Cobra CLI commands
internal/                     # Go internal packages
  timer/                      # Pure function timer logic (Tick)
  config/                     # YAML config with smart boolean merge
  state/                      # Key-value state file persistence
  idle/                       # Idle detection (ioreg on macOS)
  notify/                     # macOS notifications (osascript)
  tts/                        # Text-to-speech (say / KittenTTS / Supertonic)
  breakscreen/                # Break screen orchestration
  dashboard/                  # TUI dashboard + break activities
  ai/                         # AI CLI wrapper + history
  doctor/                     # System diagnostics
  schedule/                   # Working hours check
  launchd/                    # LaunchAgent management
  logging/                    # File-based logging with rotation
helpers/                      # Swift SPM package
  Sources/BreakScreenApp/     # Fullscreen break screen (multi-monitor)
  Sources/DashboardApp/       # Native GUI dashboard
  Sources/HelperCore/         # Shared pure logic (parsing, formatting)
  Tests/HelperCoreTests/      # Swift unit tests
Formula/                      # Homebrew formula
.github/workflows/            # CI + Release pipelines
```

### State & Logs

| File | Location |
|------|----------|
| Config | `~/.config/break-reminder/config.yaml` |
| State file | `~/.break-reminder-state` |
| Log file | `~/.break-reminder.log` |
| History | `~/.break-reminder-history.json` |
| LaunchAgent | `~/Library/LaunchAgents/com.devlikebear.break-reminder.plist` |

## 🌏 Localization

The default voice uses macOS Yuna (Korean). To change:

```bash
# List available voices
say -v '?'

# Update config
break-reminder config edit
# Change: voice: "Samantha"
```

To use KittenTTS instead:

```bash
break-reminder tts install kittentts
```

The installer auto-selects a compatible Python 3.8-3.12 interpreter when possible. Use `--bootstrap-python` to override it explicitly.

KittenTTS currently provides these built-in voices: `Bella`, `Jasper`, `Luna`, `Bruno`, `Rosie`, `Hugo`, `Kiki`, `Leo`.

You can customize the activated model or voice during install:

```bash
break-reminder tts install kittentts --voice Bella --model KittenML/kitten-tts-micro-0.8
```

To remove the managed KittenTTS environment and restore `say` defaults:

```bash
break-reminder tts uninstall kittentts
```

To use Supertonic instead:

```bash
break-reminder tts install supertonic
```

Supertonic currently provides these built-in voices: `F1`, `F2`, `F3`, `F4`, `F5`, `M1`, `M2`, `M3`, `M4`, `M5`.

The first Supertonic playback downloads the ONNX model bundle automatically (roughly 300MB). After that, Korean and English phrases run locally on-device.

You can customize the activated voice during install:

```bash
break-reminder tts install supertonic --voice F3
```

`tts_model` is still stored in config for Supertonic, but the runtime currently uses the built-in `Supertone/supertonic-2` bundle.

To remove the managed Supertonic environment and restore `say` defaults:

```bash
break-reminder tts uninstall supertonic
```

## 🤝 Contributing

Contributions are welcome! Feel free to:

- 🐛 Report bugs
- 💡 Suggest features
- 🔧 Submit pull requests

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- Inspired by the Pomodoro Technique®
- Built with ❤️ for healthier work habits

---

**Remember: Your health is more important than any deadline! 🐹💪**
