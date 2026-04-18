# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.8.0] - 2026-04-18

### Added
- **DashboardApp v2** — Native macOS dashboard fully migrated to SwiftUI with tabbed navigation (Timer / Stats / Insights)
- **Stats tab** powered by Swift Charts: weekly/monthly/all period selector, work/break bar chart, hourly focus heatmap (9–18), weekly summary cards
- **Hourly work tracking**: `DailySummary.hourly_work` field persists per-hour work minutes in `~/.break-reminder-history.json` (backward compatible)
- **AI Summary**: new `internal/insights` package builds prompts from recent history and parses claude/codex CLI responses into structured daily reports + pattern insights
- **`break-reminder insights [--refresh]`** CLI subcommand: show cached insights or force regeneration
- Auto-generation of insights on day-end (fire-and-forget goroutine when `ai_enabled: true`)
- **Theme system** (`theme: auto | dark | light` config field) with 8 color tokens via `ThemeManager`, syncs with macOS system appearance in auto mode
- **Mascot** 🐹 in status header — state-aware emoji + Korean messages (working / break / long-session warning / paused / goal-achieved)
- **Animations**: state-transition color morph, progress ring glow, tab slide transitions, mascot spring bounce, confetti on daily goal (4h)
- Window opacity dims to 55% when dashboard loses focus for clearer active/inactive signaling
- IME-independent keyboard shortcuts (physical keyCode matching — works in 한글 input mode)

### Changed
- **Minimum macOS version: 14 (Sonoma)** for Swift Charts, `onKeyPress`, `controlActiveState`, Canvas+TimelineView
- Dashboard window is now 360×600 (from 360×520) to accommodate tab bar + mascot row
- Timer tick accumulates per-hour work buckets and propagates them to `DayEndSummary` → `DailySummary`
- Insights tab buttons and refresh action shell out to `break-reminder insights --refresh`

## [0.7.1] - 2026-04-16

### Added
- `break-reminder tts set-api-key [key]` command saves the Gemini API key to the config with `0600` permissions. Supports argument, piped stdin, and refuses to prompt interactively on a TTY (so the key never lands in shell history by accident).

### Fixed
- LaunchAgent break alerts could stay silent with `tts_engine: gemini` when only `GEMINI_API_KEY` was exported in the shell, because launchd does not inherit shell env. The new command persists the key into the config file (which `ResolveAPIKey` reads first), so background timer alerts speak reliably.

## [0.7.0] - 2026-04-16

### Added
- Gemini 3.1 Flash TTS engine (`tts_engine: gemini`) with 30 prebuilt voices and native 70+ language support
- `tts_api_key` config field with `GEMINI_API_KEY` environment variable fallback via `tts.ResolveAPIKey`
- `break-reminder tts install gemini` now prints a clear message directing users to set the API key instead of running an install

### Changed
- `tts.NewSpeaker`, `tts.VoiceAvailable`, and `tts.SpeakAndWait` now take an additional `apiKey` parameter to support cloud engines
- `doctor` now surfaces a Gemini-specific install hint when the voice is unavailable

## [0.6.0] - 2026-04-11

### Added
- `break-reminder service install` now auto-registers the native menu bar app as a separate LaunchAgent when the `break-menubar` helper is installed
- Service-managed installs now keep the menu bar app running in the background after the launching terminal exits

### Changed
- Launchd management now tracks the timer daemon and menu bar app as separate agents with distinct plist files and status reporting
- `status`, `dashboard`, `doctor`, and `service status` now surface menu bar auto-start state alongside the timer agent state

### Fixed
- Homebrew/tap synchronization now preserves the full formula so helper install entries like `break-menubar` are not dropped in future releases

## [0.5.0] - 2026-04-11

### Added
- Native macOS menu bar app via `break-reminder menubar` with live progress, daily totals, and quick controls
- Deterministic `break-reminder snooze` command for ending an active break early and postponing the next one
- `break-reminder pause` and `break-reminder resume` commands that preserve active session state

### Changed
- Paused state now renders consistently across status output, TUI/GUI dashboards, and the menu bar app
- Helper-side daily totals and progress interpolation now stay day-correct across rollovers and fresh starts
- Release artifacts now ship the `break-menubar` helper alongside the existing binaries

### Fixed
- Active snooze windows no longer expire while the timer is paused
- Menu bar and dashboard helper actions no longer rewrite stale totals back into the state file

## [0.4.0] - 2026-04-09

### Added
- Minute-precision work schedule boundaries with `work_start_minute` and `work_end_minute`

### Changed
- Recovery-oriented commands remain available even when schedule config validation fails

### Fixed
- Active-break warning accounting now handles short breaks and timer edge cases more consistently
- Invalid schedule config values are rejected without clobbering the last valid config state

## [0.3.0] - 2026-03-20

### Added
- Optional `Supertonic` backend with managed venv bootstrap via `break-reminder tts install supertonic`
- `break-reminder tts uninstall <engine>` command for managed Python TTS backends
- `break-reminder tts test "<message>"` command to run blocking end-to-end speech checks

### Changed
- Doctor now suggests install commands for both `KittenTTS` and `Supertonic`
- Managed Python TTS installs now auto-select a Python 3.8-3.12 interpreter and reject incompatible runtimes earlier
- README and Homebrew caveats now document `Supertonic` setup and managed backend lifecycle

## [0.2.0] - 2026-03-20

### Added
- Optional `KittenTTS` backend with selectable engine/model/python config
- `break-reminder tts install kittentts` command to bootstrap a managed venv

### Changed
- Doctor output now guides users to install `KittenTTS` when selected but unavailable
- Homebrew caveats and README now document optional `KittenTTS` setup

## [0.1.1] - 2026-03-19

### Fixed
- Release workflow: use env var for secrets in if-condition
- Release workflow: sync Homebrew tap repo with PAT

### Changed
- Add VERSION file for centralized version management

## [0.1.0] - 2026-03-19

### Added
- **Go rewrite** — Complete rewrite from bash to Go + Swift
- **Fullscreen break screen** — Swift AppKit overlay with multi-monitor support
- **Native GUI dashboard** — `break-reminder dashboard --gui` for macOS native UI
- **TUI dashboard** — Bubbletea real-time dashboard with progress bars and daily stats
- **Guided break activities** — Eye exercise (20-20-20), stretching, box breathing, walk timer
- **AI integration** — Claude/Codex CLI for productivity analysis (`ai summary`, `ai suggest`, `ai configure`)
- **Daily history tracking** — Automatic day-end summary persistence for AI analysis
- **Gap detection** — Skip sleep/wake gaps to prevent inflated work time
- **Today's stats display** — Work/break stats shown on break screen, dashboard, and status command
- **Homebrew support** — `brew install devlikebear/tap/break-reminder`
- **GitHub Actions CI/CD** — Build + test on push/PR, automated release with Homebrew tap update
- **System diagnostics** — `break-reminder doctor` checks all components
- **YAML configuration** — `~/.config/break-reminder/config.yaml` with smart boolean merging
- **Hot-reload config** — Daemon and dashboard modes pick up config changes without restart
- **Platform abstraction** — Build tags for macOS-specific features (idle, notify, TTS)
- **Service management** — `break-reminder service install/uninstall/start/stop/status`

### Architecture
- Pure function `timer.Tick()` — no side effects, fully testable
- Swift SPM package with shared HelperCore library
- Go ↔ Swift communication via CLI subprocess + arguments
- Comprehensive test suite: Go unit tests + Swift HelperCore tests

---

## Legacy (bash script)

## [1.2.0] - 2025-01-29

### Security
- Replace `source` command with safe key-value parsing in `read_state()` to prevent potential code injection

## [1.1.0] - 2025-01-29

### Added
- TUI Dashboard with real-time progress bars and stats (`dashboard` command)
- Smart scheduling - only active during configured working hours/days
- Daily statistics tracking (work time, break time, ratio)
- Built-in `install` and `uninstall` commands for launchd agent management
- Log rotation (configurable max lines)
- Installation status display in status/dashboard

### Changed
- Improved status output with more detailed information

## [1.0.1] - 2025-01-28

### Fixed
- SIGPIPE error (exit 141) when running in certain terminal environments

## [1.0.0] - 2025-01-28

### Added
- Initial release
- Activity detection via keyboard/mouse monitoring
- 50/10 work/break cycle with configurable durations
- macOS native notifications with sound
- Text-to-speech announcements
- Natural break detection (auto-reset on extended idle)
- Break enforcement warnings
- launchd integration for auto-start
- State persistence across restarts
