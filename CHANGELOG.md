# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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
