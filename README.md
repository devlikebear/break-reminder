# 🐹 Break Reminder

**A smart break reminder for macOS that monitors your activity and enforces healthy work habits.**

Work 50 minutes → Rest 10 minutes → Repeat!

![macOS](https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Shell](https://img.shields.io/badge/shell-bash-green.svg)

## ✨ Features

- **🖱️ Activity Detection** - Monitors keyboard/mouse usage to track actual work time
- **⏰ Smart Timer** - 50 min work / 10 min break cycle (configurable)
- **🔔 Notifications** - Visual + voice alerts (uses macOS native TTS)
- **🧘 Natural Break Detection** - Automatically resets if you're already taking a break
- **🚫 Break Enforcement** - Warns you if you try to work during break time
- **🚀 Auto-start** - Runs automatically on login via launchd

## 📦 Installation

### Quick Install

```bash
# Clone the repository
git clone https://github.com/devlikebear/break-reminder.git
cd break-reminder

# Run installer
./install.sh
```

### Manual Install

```bash
# Clone the repository
git clone https://github.com/devlikebear/break-reminder.git
cd break-reminder

# Make executable
chmod +x break-reminder.sh

# Copy to your preferred location
cp break-reminder.sh ~/Scripts/

# Install launch agent (auto-start on login)
cp com.user.break-reminder.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.user.break-reminder.plist
```

## 🚀 Usage

### Commands

```bash
# Check current status
break-reminder status

# Manually reset the timer
break-reminder reset

# Run a single check (used by launchd)
break-reminder check

# Run as foreground daemon (for testing)
break-reminder daemon
```

### Example Output

```
🐹 Break Reminder Status
========================
Mode: work
Work time: 32min / 50min
Current idle: 5sec
```

## ⚙️ Configuration

Edit the variables at the top of `break-reminder.sh`:

```bash
WORK_DURATION=$((50 * 60))    # Work duration in seconds (default: 50 min)
BREAK_DURATION=$((10 * 60))   # Break duration in seconds (default: 10 min)
IDLE_THRESHOLD=120            # Seconds of idle to count as "not working" (default: 2 min)
```

### Optional: Screen Lock on Break

Uncomment this line in the script to force screen lock when break starts:

```bash
# lock_screen
```

## 🔧 Managing the Service

```bash
# Stop the service
launchctl unload ~/Library/LaunchAgents/com.user.break-reminder.plist

# Start the service
launchctl load ~/Library/LaunchAgents/com.user.break-reminder.plist

# Check if running
launchctl list | grep break-reminder
```

## 📁 Files

| File | Description |
|------|-------------|
| `break-reminder.sh` | Main script |
| `com.user.break-reminder.plist` | launchd configuration |
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

# Edit the script and change the voice
speak_message() {
    say -v "Samantha" "$1" &  # English voice
}
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
