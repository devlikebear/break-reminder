package timetools

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestStoreRoundTripRevisionAndUnknownFields(t *testing.T) {
	store := NewStore(t.TempDir())
	s, err := store.Load()
	if err != nil || s.Revision != 0 {
		t.Fatal(s, err)
	}
	s, err = store.Update(nil, func(s Snapshot) (Snapshot, error) {
		return Apply(s, Command{Kind: "start", ID: "a", DurationMS: 1000}, 1000)
	})
	if err != nil || s.Revision != 1 {
		t.Fatal(s, err)
	}
	info, _ := os.Stat(store.Path)
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
	rev := uint64(0)
	if _, err = store.Update(&rev, func(s Snapshot) (Snapshot, error) { return s, nil }); ErrorCode(err) != "conflict" {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(store.Path)
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	raw["future"] = json.RawMessage(`{"keep":true}`)
	data, _ = json.Marshal(raw)
	_ = os.WriteFile(store.Path, data, 0600)
	_, err = store.Update(nil, func(s Snapshot) (Snapshot, error) { return Apply(s, Command{Kind: "pause", ID: "a"}, 1100) })
	if err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(store.Path)
	_ = json.Unmarshal(data, &raw)
	if string(raw["future"]) != `{"keep":true}` {
		t.Fatal(string(data))
	}
}
func TestStoreRejectsCorruptAndFutureWithoutWriting(t *testing.T) {
	for _, data := range []string{`oops`, `{"schema_version":99}`} {
		store := NewStore(t.TempDir())
		_ = os.WriteFile(store.Path, []byte(data), 0600)
		if _, err := store.Update(nil, func(s Snapshot) (Snapshot, error) { return s, nil }); err == nil {
			t.Fatal("expected failure")
		}
		got, _ := os.ReadFile(store.Path)
		if string(got) != data {
			t.Fatal("overwritten")
		}
	}
}
func TestStoreConcurrentWritersAndNoOp(t *testing.T) {
	store := NewStore(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Update(nil, func(s Snapshot) (Snapshot, error) { s.Extra["test"] = json.RawMessage(`true`); return s, nil })
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	s, err := store.Load()
	if err != nil || s.Revision != 1 {
		t.Fatal(s, err)
	}
	before, _ := os.Stat(store.Path)
	_, err = store.Update(nil, func(s Snapshot) (Snapshot, error) { return s, nil })
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(store.Path)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("no-op rewrote file")
	}
	if matches, _ := filepath.Glob(store.Path + ".tmp-*"); len(matches) != 0 {
		t.Fatal(matches)
	}
}

func TestStoreSeparateProcessesSerialize(t *testing.T) {
	if dir := os.Getenv("TIMETOOLS_TEST_CHILD_DIR"); dir != "" {
		store := NewStore(dir)
		for i := 0; i < 10; i++ {
			_, err := store.Update(nil, func(s Snapshot) (Snapshot, error) {
				var count int
				_ = json.Unmarshal(s.Extra["counter"], &count)
				s.Extra["counter"], _ = json.Marshal(count + 1)
				return s, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	dir := t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var commands []*exec.Cmd
	for i := 0; i < 4; i++ {
		cmd := exec.Command(binary, "-test.run=^TestStoreSeparateProcessesSerialize$")
		cmd.Env = append(os.Environ(), "TIMETOOLS_TEST_CHILD_DIR="+dir)
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		if err = cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewStore(dir).Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Revision != 40 || string(s.Extra["counter"]) != "40" {
		t.Fatal(s.Revision, string(s.Extra["counter"]))
	}
}

func TestStoreCallbackFailurePreservesOriginal(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Update(nil, func(s Snapshot) (Snapshot, error) {
		return Apply(s, Command{Kind: "start", ID: "a", DurationMS: 10000}, 1000)
	})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.Path)
	_, err = store.Update(nil, func(s Snapshot) (Snapshot, error) {
		s.Countdown.Label = "mutated"
		return s, errors.New("save rejected")
	})
	if err == nil {
		t.Fatal("wanted failure")
	}
	after, _ := os.ReadFile(store.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed operation modified original")
	}
}
