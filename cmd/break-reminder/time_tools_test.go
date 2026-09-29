package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/timetools"
)

func timeCommand(t *testing.T, dir string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetArgs(append([]string{"time-tools", "--data-dir", dir}, args...))
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	err := cmd.Execute()
	return out.Bytes(), err
}
func fakeWorker(t *testing.T, dir string) {
	t.Helper()
	data, _ := json.Marshal(timetools.Runtime{SchemaVersion: 1, Heartbeat: time.Now().UnixMilli(), PID: os.Getpid(), Ready: true})
	if err := os.WriteFile(filepath.Join(dir, "time-tools-runtime.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestTimeToolsIndependentOfInvalidWorkConfig(t *testing.T) {
	old := loadConfig
	t.Cleanup(func() { loadConfig = old })
	loadConfig = func() (config.Config, error) { return config.Config{}, errors.New("invalid work config") }
	dir := t.TempDir()
	out, err := timeCommand(t, dir, "status", "--json")
	if err != nil || !json.Valid(out) {
		t.Fatal(string(out), err)
	}
	fakeWorker(t, dir)
	out, err = timeCommand(t, dir, "timer", "start", "--duration", "15m", "--label", "코드 리뷰", "--json")
	if err != nil || !json.Valid(out) {
		t.Fatal(string(out), err)
	}
	var response struct{ Snapshot timetools.Snapshot }
	_ = json.Unmarshal(out, &response)
	if response.Snapshot.Countdown.Label != "코드 리뷰" {
		t.Fatal(string(out))
	}
	out, err = timeCommand(t, dir, "timer", "pause", "--id", response.Snapshot.Countdown.ID, "--if-revision", "0", "--json")
	if err == nil || !json.Valid(out) {
		t.Fatal("stale revision accepted", string(out), err)
	}
	s, _ := timetools.NewStore(dir).Load()
	if s.Countdown.Phase != "running" {
		t.Fatal(s)
	}
}
func TestTimeToolsStartRequiresWorkerAndRejectsInvalidInput(t *testing.T) {
	dir := t.TempDir()
	out, err := timeCommand(t, dir, "timer", "start", "--duration", "1s", "--json")
	if err == nil || !bytes.Contains(out, []byte("worker_unavailable")) {
		t.Fatal(string(out), err)
	}
	if _, err = os.Stat(timetools.NewStore(dir).Path); !os.IsNotExist(err) {
		t.Fatal("start wrote data without worker")
	}
	fakeWorker(t, dir)
	for _, duration := range []string{"0", "abc", "500ms", "25h"} {
		out, err = timeCommand(t, dir, "timer", "start", "--duration", duration, "--json")
		if err == nil || !json.Valid(out) {
			t.Fatal(duration, string(out), err)
		}
	}
}
