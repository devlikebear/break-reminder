package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/enginelock"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestEngineStdioSmoke(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BREAK_REMINDER_HOME", root)
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en")

	inR, inW := io.Pipe()
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- runEngine(context.Background(), inR, &out, time.Hour) }()

	io.WriteString(inW, `{"jsonrpc":"2.0","id":1,"method":"engine.ping"}`+"\n")
	io.WriteString(inW, `{"jsonrpc":"2.0","id":2,"method":"session.start"}`+"\n")
	io.WriteString(inW, `{"jsonrpc":"2.0","id":3,"method":"status.get"}`+"\n")

	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(out.String(), `"id":3`) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !enginelock.Held(filepath.Join(root, "engine.lock")) {
		t.Error("engine lock not held while serving")
	}
	inW.Close()
	if err := <-done; err != nil {
		t.Fatalf("runEngine() = %v", err)
	}
	if enginelock.Held(filepath.Join(root, "engine.lock")) {
		t.Error("engine lock still held after exit")
	}

	ids := map[float64]map[string]any{}
	sawStatus := false
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("stdout must be JSON only, got %q", line)
		}
		if id, ok := m["id"].(float64); ok {
			ids[id] = m
		}
		if m["method"] == "status" {
			sawStatus = true
		}
	}
	if len(ids) != 3 {
		t.Fatalf("responses = %v", ids)
	}
	state := ids[3]["result"].(map[string]any)["state"].(map[string]any)
	if state["session_state"] != "active" {
		t.Fatalf("session not active after session.start: %v", state)
	}
	if !sawStatus {
		t.Error("no status notification")
	}
}

func TestEngineRefusesSecondInstance(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BREAK_REMINDER_HOME", root)
	lock, err := enginelock.Acquire(filepath.Join(root, "engine.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := runEngine(context.Background(), strings.NewReader(""), io.Discard, time.Hour); err == nil {
		t.Fatal("second engine started while lock held")
	}
}
