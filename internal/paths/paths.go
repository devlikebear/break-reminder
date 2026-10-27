// Package paths decides where every on-disk file of break-reminder lives.
//
// Without overrides the legacy layout under the user's home directory is kept
// so existing installations keep working. When BREAK_REMINDER_HOME is set to
// an absolute directory, every file lives directly under it; the menu bar app
// uses this to point the engine at a temp dir (development) or at the sandbox
// container (App Store build).
package paths

import (
	"os"
	"path/filepath"
)

// EnvHome names the environment variable that relocates all data files.
const EnvHome = "BREAK_REMINDER_HOME"

func overrideRoot() string {
	root := os.Getenv(EnvHome)
	if root == "" || !filepath.IsAbs(root) {
		return ""
	}
	return filepath.Clean(root)
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

// Overridden reports whether BREAK_REMINDER_HOME is in effect.
func Overridden() bool { return overrideRoot() != "" }

func pick(override, legacy string) string {
	if root := overrideRoot(); root != "" {
		return filepath.Join(root, override)
	}
	return filepath.Join(userHome(), legacy)
}

// ConfigDir holds config.yaml and time-tools data.
func ConfigDir() string {
	if root := overrideRoot(); root != "" {
		return root
	}
	return filepath.Join(userHome(), ".config", "break-reminder")
}

// ConfigFile is the YAML settings file.
func ConfigFile() string { return filepath.Join(ConfigDir(), "config.yaml") }

// StateFile is the key=value timer state file.
func StateFile() string { return pick("state", ".break-reminder-state") }

// TimeToolsDir holds time-tools.json.
func TimeToolsDir() string { return ConfigDir() }

// HistoryFile is the AI daily history file.
func HistoryFile() string { return pick("history.json", ".break-reminder-history.json") }

// InsightsFile is the AI insights report.
func InsightsFile() string { return pick("insights.json", ".break-reminder-insights.json") }

// LogFile is the rotating log file.
func LogFile() string { return pick("break-reminder.log", ".break-reminder.log") }

// EngineLockFile is held by a running engine so that launchd `check` and the
// time-tools worker stay out of its way.
func EngineLockFile() string { return filepath.Join(ConfigDir(), "engine.lock") }

// CalendarCacheFile keeps the last calendar snapshot sent by the app.
func CalendarCacheFile() string { return filepath.Join(ConfigDir(), "calendar-cache.json") }

// AppUsageDir holds one JSON file of per-app foreground time per day.
func AppUsageDir() string { return filepath.Join(ConfigDir(), "app-usage") }

// MCPDiscoveryFile tells the stdio bridge where the engine's MCP endpoint is.
func MCPDiscoveryFile() string { return filepath.Join(ConfigDir(), "mcp.json") }

// AssistantDir holds one file per day with the briefing and retrospective.
func AssistantDir() string { return filepath.Join(ConfigDir(), "assistant") }
