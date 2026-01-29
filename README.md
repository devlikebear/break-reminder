# 🐹 Break Reminder

**A smart break reminder for macOS that monitors your activity and enforces healthy work habits.**

Work 50 minutes → Rest 10 minutes → Repeat!

![macOS](https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Shell](https://img.shields.io/badge/shell-bash-green.svg)

## ✨ Features

- **🖱️ Activity Detection** - Monitors keyboard/mouse usage to track actual work time
- **⏰ Smart Timer** - 50 min work / 10 min break cycle (configurable)
- **📊 TUI Dashboard** - Real-time progress bars and daily statistics
- **🗓️ Smart Scheduling** - Only active during configured working hours/days
- **📈 Daily Stats** - Track your work/break time and ratio
- **🔔 Notifications** - Visual + voice alerts (uses macOS native TTS)
- **🧘 Natural Break Detection** - Automatically resets if you're already taking a break
- **🚫 Break Enforcement** - Warns you if you try to work during break time
- **🚀 Auto-start** - Runs automatically on login via launchd
- **🔒 Secure** - Safe state file parsing with input validation

## 📦 Installation

### Quick Install

```bash
# Clone the repository
git clone https://github.com/devlikebear/break-reminder.git
cd break-reminder

# Run installer
./install.sh
```

### Using Built-in Command

```bash
# Install launchd agent (auto-start on login)
./break-reminder.sh install

# Uninstall launchd agent
./break-reminder.sh uninstall
```

## 🚀 Usage

### Commands

```bash
# Real-time TUI dashboard
break-reminder dashboard

# Check current status
break-reminder status

# Manually reset the timer
break-reminder reset

# Install as launchd agent
break-reminder install

# Uninstall launchd agent
break-reminder uninstall

# Run a single check (used by launchd)
break-reminder check

# Run as foreground daemon (for testing)
break-reminder daemon

# Show help
break-reminder help
```

### Dashboard

```
🐹 Break Reminder Dashboard (Press 'q' to quit)
==================================================
System: Installed & Running
Status: WORKING
Idle: 5s / Limit: 120s

Session Work: [################--------------] 53% (26 / 50 min)

Daily Statistics:
  Work: 180 min
  Rest: 30 min
  Ratio: [####################] 86%

Recent Logs:
--------------------------------------------------
  [2025-01-29 14:30:00] Working... 26min elapsed (24min remaining)
--------------------------------------------------
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
Daily Stats: Work 180min / Break 30min
Current idle: 5sec
```

## ⚙️ Configuration

Edit the variables at the top of `break-reminder.sh`:

```bash
# Timer Settings
WORK_DURATION=$((50 * 60))    # Work duration (default: 50 min)
BREAK_DURATION=$((10 * 60))   # Break duration (default: 10 min)
IDLE_THRESHOLD=120            # Idle threshold (default: 2 min)
NATURAL_BREAK_THRESHOLD=300   # Auto-reset threshold (default: 5 min)

# Smart Scheduling
WORK_DAYS="1 2 3 4 5"         # Working days (1=Mon, 7=Sun)
WORK_START_HOUR=9             # Start hour (24h format)
WORK_END_HOUR=18              # End hour (24h format)

# Log Settings
MAX_LOG_LINES=1000            # Log rotation threshold

# Voice Settings
VOICE="Yuna"                  # TTS voice (run `say -v '?'` for options)
```

### Optional: Screen Lock on Break

Uncomment this line in the script to force screen lock when break starts:

```bash
# lock_screen
```

## 🔧 Managing the Service

```bash
# Check installation status
break-reminder status

# Install/Uninstall via built-in commands
break-reminder install
break-reminder uninstall

# Or manually via launchctl
launchctl unload ~/Library/LaunchAgents/com.devlikebear.break-reminder.plist
launchctl load ~/Library/LaunchAgents/com.devlikebear.break-reminder.plist

# Check if running
launchctl list | grep break-reminder
```

## 📁 Files

| File | Description |
|------|-------------|
| `break-reminder.sh` | Main script |
| `com.user.break-reminder.plist` | launchd configuration (template) |
| `install.sh` | Installation script |
| `uninstall.sh` | Uninstallation script |

### State & Logs

| File | Location |
|------|----------|
| State file | `~/.break-reminder-state` |
| Log file | `~/.break-reminder.log` |

## 🌏 Localization

The default voice uses macOS Yuna (Korean). To change the voice:

```bash
# List available voices
say -v '?'

# Edit the VOICE variable in the script
VOICE="Samantha"  # English voice
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
