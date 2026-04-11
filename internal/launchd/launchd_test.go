package launchd

import (
	"path/filepath"
	"strings"
	"testing"
)

func stubUserHomeDir(t *testing.T, home string) {
	t.Helper()

	orig := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() {
		userHomeDir = orig
	})
}

func TestPlistPathsUseDistinctLabels(t *testing.T) {
	home := t.TempDir()
	stubUserHomeDir(t, home)

	if got, want := PlistPath(), filepath.Join(home, "Library", "LaunchAgents", Label+".plist"); got != want {
		t.Fatalf("PlistPath() = %q, want %q", got, want)
	}

	if got, want := MenuBarPlistPath(), filepath.Join(home, "Library", "LaunchAgents", MenuBarLabel+".plist"); got != want {
		t.Fatalf("MenuBarPlistPath() = %q, want %q", got, want)
	}
}

func TestGenerateTimerPlistRunsPeriodicCheck(t *testing.T) {
	plist := generateTimerPlist("/tmp/break-reminder")

	for _, want := range []string{
		"<string>" + Label + "</string>",
		"<string>/tmp/break-reminder</string>",
		"<string>check</string>",
		"<key>StartInterval</key>",
		"<integer>60</integer>",
		"<key>RunAtLoad</key>",
		"<string>/tmp/break-reminder.out</string>",
		"<string>/tmp/break-reminder.err</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("generateTimerPlist() missing %q in plist:\n%s", want, plist)
		}
	}
}

func TestGenerateMenuBarPlistKeepsAccessoryAppAlive(t *testing.T) {
	plist := generateMenuBarPlist("/tmp/break-menubar")

	for _, want := range []string{
		"<string>" + MenuBarLabel + "</string>",
		"<string>/tmp/break-menubar</string>",
		"<key>RunAtLoad</key>",
		"<key>KeepAlive</key>",
		"<key>LimitLoadToSessionType</key>",
		"<string>Aqua</string>",
		"<string>/tmp/break-reminder-menubar.out</string>",
		"<string>/tmp/break-reminder-menubar.err</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("generateMenuBarPlist() missing %q in plist:\n%s", want, plist)
		}
	}
}
