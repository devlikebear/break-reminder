package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyLayoutWithoutOverride(t *testing.T) {
	t.Setenv(EnvHome, "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]string{
		"config dir": {ConfigDir(), filepath.Join(home, ".config", "break-reminder")},
		"config":     {ConfigFile(), filepath.Join(home, ".config", "break-reminder", "config.yaml")},
		"state":      {StateFile(), filepath.Join(home, ".break-reminder-state")},
		"time tools": {TimeToolsDir(), filepath.Join(home, ".config", "break-reminder")},
		"history":    {HistoryFile(), filepath.Join(home, ".break-reminder-history.json")},
		"insights":   {InsightsFile(), filepath.Join(home, ".break-reminder-insights.json")},
		"log":        {LogFile(), filepath.Join(home, ".break-reminder.log")},
		"lock":       {EngineLockFile(), filepath.Join(home, ".config", "break-reminder", "engine.lock")},
		"calendar":   {CalendarCacheFile(), filepath.Join(home, ".config", "break-reminder", "calendar-cache.json")},
		"app usage":  {AppUsageDir(), filepath.Join(home, ".config", "break-reminder", "app-usage")},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	if Overridden() {
		t.Error("Overridden() = true without env")
	}
}

func TestOverrideRootsEverything(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvHome, root)
	cases := map[string][2]string{
		"config dir": {ConfigDir(), root},
		"config":     {ConfigFile(), filepath.Join(root, "config.yaml")},
		"state":      {StateFile(), filepath.Join(root, "state")},
		"time tools": {TimeToolsDir(), root},
		"history":    {HistoryFile(), filepath.Join(root, "history.json")},
		"insights":   {InsightsFile(), filepath.Join(root, "insights.json")},
		"log":        {LogFile(), filepath.Join(root, "break-reminder.log")},
		"lock":       {EngineLockFile(), filepath.Join(root, "engine.lock")},
		"calendar":   {CalendarCacheFile(), filepath.Join(root, "calendar-cache.json")},
		"app usage":  {AppUsageDir(), filepath.Join(root, "app-usage")},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	if !Overridden() {
		t.Error("Overridden() = false with env")
	}
}

func TestRelativeOverrideIsIgnored(t *testing.T) {
	t.Setenv(EnvHome, "relative/dir")
	home, _ := os.UserHomeDir()
	if got := StateFile(); got != filepath.Join(home, ".break-reminder-state") {
		t.Fatalf("relative override should be ignored, got %q", got)
	}
}
