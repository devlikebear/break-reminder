package workcontext

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUsageAddAccumulatesPerDay(t *testing.T) {
	u := UsageStore{Dir: t.TempDir()}
	day := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	if err := u.Add(day, []AppUsage{{BundleID: "com.apple.Safari", Name: "Safari", Seconds: 60}, {BundleID: "com.microsoft.VSCode", Name: "Code", Seconds: 30}}); err != nil {
		t.Fatal(err)
	}
	if err := u.Add(day, []AppUsage{{BundleID: "com.microsoft.VSCode", Name: "Visual Studio Code", Seconds: 90}}); err != nil {
		t.Fatal(err)
	}
	got, err := u.Day(day)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].BundleID != "com.microsoft.VSCode" || got[0].Seconds != 120 || got[0].Name != "Visual Studio Code" {
		t.Fatalf("day = %+v", got)
	}
	if top := Top(got, 1); len(top) != 1 || top[0].Seconds != 120 {
		t.Fatalf("top = %+v", top)
	}
	other, _ := u.Day(day.AddDate(0, 0, 1))
	if len(other) != 0 {
		t.Fatalf("next day = %+v", other)
	}
}

func TestUsageAddRejectsBadIncrements(t *testing.T) {
	u := UsageStore{Dir: t.TempDir()}
	day := time.Now()
	for _, bad := range []AppUsage{{BundleID: "", Seconds: 10}, {BundleID: "x", Seconds: -1}, {BundleID: "x", Seconds: 3601}} {
		if err := u.Add(day, []AppUsage{bad}); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

func TestUsagePruneAndDeleteAll(t *testing.T) {
	dir := t.TempDir()
	u := UsageStore{Dir: dir}
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, time.Local)
	for _, d := range []int{0, -29, -30, -45} {
		if err := u.Add(today.AddDate(0, 0, d), []AppUsage{{BundleID: "a", Name: "A", Seconds: 1}}); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0o600)
	if err := u.Prune(today, 30); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := map[string]bool{"2026-10-20.json": true, "2026-09-21.json": true, "notes.txt": true}
	if len(names) != len(want) {
		t.Fatalf("after prune: %v", names)
	}
	for _, n := range names {
		if !want[n] {
			t.Fatalf("unexpected file kept: %s (all: %v)", n, names)
		}
	}
	if err := u.DeleteAll(); err != nil {
		t.Fatal(err)
	}
	if got, _ := u.Day(today); len(got) != 0 {
		t.Fatalf("after delete: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal("DeleteAll must only remove usage files")
	}
}
