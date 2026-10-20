package enginelock

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeldReflectsAcquireAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "engine.lock")
	if Held(path) {
		t.Fatal("Held() before acquire = true")
	}
	lock, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	if !Held(path) {
		t.Fatal("Held() while locked = false")
	}
	if _, err := Acquire(path); err == nil {
		t.Fatal("second Acquire succeeded while first holds the lock")
	}
	lock.Release()
	if Held(path) {
		t.Fatal("Held() after release = true")
	}
}

// The lock must be visible across processes, which is how launchd `check`
// meets a running engine.
func TestHeldAcrossProcesses(t *testing.T) {
	if os.Getenv("ENGINELOCK_HELPER") != "" {
		if Held(os.Getenv("ENGINELOCK_HELPER")) {
			os.Stdout.WriteString("held")
		} else {
			os.Stdout.WriteString("free")
		}
		os.Exit(0)
	}
	path := filepath.Join(t.TempDir(), "engine.lock")
	lock, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	cmd := exec.Command(os.Args[0], "-test.run=TestHeldAcrossProcesses")
	cmd.Env = append(os.Environ(), "ENGINELOCK_HELPER="+path)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "held") {
		t.Fatalf("child saw %q, want held", out)
	}
}
