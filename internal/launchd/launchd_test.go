package launchd

import (
	"os"
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
	if got, want := UpdaterPlistPath(), filepath.Join(home, "Library", "LaunchAgents", UpdaterLabel+".plist"); got != want {
		t.Fatalf("UpdaterPlistPath() = %q, want %q", got, want)
	}
}

func TestUpdaterStatusIsNotInstalledWithoutPlist(t *testing.T) {
	stubUserHomeDir(t, t.TempDir())
	if got := UpdaterStatus(); got != "Not Installed" {
		t.Fatalf("UpdaterStatus() = %q, want Not Installed", got)
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

func TestGenerateUpdaterPlistRunsDailyWithoutRunAtLoad(t *testing.T) {
	home := t.TempDir()
	stubUserHomeDir(t, home)
	plist := generateUpdaterPlist("/opt/homebrew/bin/break-reminder")

	for _, want := range []string{
		"<string>" + UpdaterLabel + "</string>",
		"<string>/opt/homebrew/bin/break-reminder</string>",
		"<string>update</string>",
		"<string>--automatic</string>",
		"<key>StartCalendarInterval</key>",
		"<key>Hour</key>",
		"<integer>4</integer>",
		"<key>Minute</key>",
		"<integer>0</integer>",
		"<string>" + filepath.Join(home, "Library", "Logs", "break-reminder-updater.out") + "</string>",
		"<string>" + filepath.Join(home, "Library", "Logs", "break-reminder-updater.err") + "</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("generateUpdaterPlist() missing %q in plist:\n%s", want, plist)
		}
	}
	if strings.Contains(plist, "<key>RunAtLoad</key>") {
		t.Fatalf("generateUpdaterPlist() must not run immediately on install:\n%s", plist)
	}
	if strings.Contains(plist, "/tmp/break-reminder-updater") {
		t.Fatalf("generateUpdaterPlist() uses shared /tmp logs:\n%s", plist)
	}
}

func TestGenerateUpdaterPlistEscapesBinaryPath(t *testing.T) {
	plist := generateUpdaterPlist("/tmp/A&B<bin>/break-reminder")
	if !strings.Contains(plist, "<string>/tmp/A&amp;B&lt;bin&gt;/break-reminder</string>") {
		t.Fatalf("generateUpdaterPlist() did not XML-escape binary path:\n%s", plist)
	}
}

func TestInstallUpdaterWritesAndLoadsPlist(t *testing.T) {
	home := t.TempDir()
	stubUserHomeDir(t, home)
	oldLoadJob := loadJobForInstall
	var loadedPath string
	loadJobForInstall = func(path string) error {
		loadedPath = path
		return nil
	}
	t.Cleanup(func() { loadJobForInstall = oldLoadJob })

	if err := InstallUpdater("/opt/homebrew/bin/break-reminder"); err != nil {
		t.Fatalf("InstallUpdater() error = %v", err)
	}
	if loadedPath != UpdaterPlistPath() {
		t.Fatalf("loaded path = %q, want %q", loadedPath, UpdaterPlistPath())
	}
	contents, err := os.ReadFile(UpdaterPlistPath())
	if err != nil {
		t.Fatalf("read updater plist: %v", err)
	}
	if !strings.Contains(string(contents), "<string>/opt/homebrew/bin/break-reminder</string>") {
		t.Fatalf("updater plist uses wrong binary:\n%s", contents)
	}
}
