package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	Log(path, "hello world")
	Log(path, "second line")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if !strings.Contains(lines[0], "hello world") {
		t.Errorf("line 0 = %q, want to contain 'hello world'", lines[0])
	}
	if !strings.HasPrefix(lines[0], "[") {
		t.Errorf("line 0 should start with timestamp bracket, got %q", lines[0])
	}
}

func TestRotate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	// Write 10 lines
	for i := 0; i < 10; i++ {
		Log(path, "line")
	}

	// Rotate to 5 lines
	Rotate(path, 5)

	lines := readLines(t, path)
	if len(lines) != 5 {
		t.Errorf("got %d lines after rotate, want 5", len(lines))
	}
}

func TestRotateNoOpWhenUnderLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	Log(path, "only one")
	Rotate(path, 100)

	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Errorf("got %d lines, want 1 (unchanged)", len(lines))
	}
}

func TestRotateMissingFile(t *testing.T) {
	// Should not panic
	Rotate(filepath.Join(t.TempDir(), "nonexistent.log"), 10)
}

func TestTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	for i := 0; i < 10; i++ {
		Log(path, "line")
	}

	got := Tail(path, 3)
	if len(got) != 3 {
		t.Errorf("Tail(3) returned %d lines, want 3", len(got))
	}
}

func TestTailFewerLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	Log(path, "only one")

	got := Tail(path, 5)
	if len(got) != 1 {
		t.Errorf("Tail(5) returned %d lines, want 1", len(got))
	}
}

func TestTailMissingFile(t *testing.T) {
	got := Tail(filepath.Join(t.TempDir(), "nope.log"), 5)
	if got != nil {
		t.Errorf("Tail on missing file = %v, want nil", got)
	}
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
