package launchd

import (
	"os"
	"strings"
	"testing"
)

func TestTimeToolsPlistAndInstallWithoutMenuBar(t *testing.T) {
	stubUserHomeDir(t, t.TempDir())
	old := loadJobForInstall
	oldRemove := removeRuntimeJob
	t.Cleanup(func() { loadJobForInstall = old; removeRuntimeJob = oldRemove })
	loaded := []string{}
	loadJobForInstall = func(path string) error { loaded = append(loaded, path); return nil }
	removeRuntimeJob = func(string) error { return nil }
	if _, err := Install("/opt/homebrew/bin/break-reminder", ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(TimeToolsPlistPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"time-tools", "run", "KeepAlive", "Aqua", "ThrottleInterval"} {
		if !strings.Contains(string(data), want) {
			t.Fatal(want, string(data))
		}
	}
	if len(loaded) != 2 || loaded[1] != TimeToolsPlistPath() {
		t.Fatal(loaded)
	}
}
func TestMigrateRuntimePreservesStoppedJobs(t *testing.T) {
	stubUserHomeDir(t, t.TempDir())
	old := loadJobForInstall
	t.Cleanup(func() { loadJobForInstall = old })
	loaded := []string{}
	loadJobForInstall = func(p string) error { loaded = append(loaded, p); return nil }
	state := RuntimeInstallation{Timer: JobInstallation{Installed: true}, TimeTools: JobInstallation{Installed: true}}
	if err := MigrateRuntime("/bin/br", "", state); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatal("restarted stopped runtime", loaded)
	}
	if _, err := os.Stat(TimeToolsPlistPath()); err != nil {
		t.Fatal(err)
	}
	state.TimeTools.Loaded = true
	if err := MigrateRuntime("/bin/br", "", state); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0] != TimeToolsPlistPath() {
		t.Fatal(loaded)
	}
	if _, err := os.Stat(UpdaterPlistPath()); !os.IsNotExist(err) {
		t.Fatal("migration changed updater")
	}
}
