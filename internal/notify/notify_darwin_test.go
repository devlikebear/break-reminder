//go:build darwin

package notify

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// These tests replace package-level dependency hooks. Keep them serial; do not
// add t.Parallel without first moving the hooks onto per-test notifier values.

func TestResolveTerminalNotifierUsesPATHFirst(t *testing.T) {
	oldLookPath := lookPath
	t.Cleanup(func() { lookPath = oldLookPath })

	lookPath = func(name string) (string, error) {
		if name != "terminal-notifier" {
			t.Fatalf("lookPath(%q), want terminal-notifier", name)
		}
		return "/custom/bin/terminal-notifier", nil
	}

	got, err := resolveTerminalNotifier()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/bin/terminal-notifier" {
		t.Fatalf("resolveTerminalNotifier() = %q, want PATH result", got)
	}
}

func TestResolveTerminalNotifierFindsHomebrewPathForLaunchd(t *testing.T) {
	oldLookPath := lookPath
	oldFileExists := fileExists
	t.Cleanup(func() {
		lookPath = oldLookPath
		fileExists = oldFileExists
	})

	lookPath = func(string) (string, error) { return "", errors.New("not in launchd PATH") }
	fileExists = func(path string) bool { return path == "/opt/homebrew/bin/terminal-notifier" }

	got, err := resolveTerminalNotifier()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/opt/homebrew/bin/terminal-notifier" {
		t.Fatalf("resolveTerminalNotifier() = %q, want Homebrew path", got)
	}
}

func TestResolveTerminalNotifierReturnsActionableErrorWhenMissing(t *testing.T) {
	oldLookPath := lookPath
	oldFileExists := fileExists
	t.Cleanup(func() {
		lookPath = oldLookPath
		fileExists = oldFileExists
	})

	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	fileExists = func(string) bool { return false }

	_, err := resolveTerminalNotifier()
	if err == nil {
		t.Fatal("resolveTerminalNotifier() error = nil, want missing dependency error")
	}
	if !strings.Contains(err.Error(), "brew install terminal-notifier") {
		t.Fatalf("error = %q, want installation guidance", err)
	}
}

func TestNotificationArgsHaveNoClickAction(t *testing.T) {
	got := notificationArgs("Break Over!", "Back to work!", "Hero")
	want := []string{
		"-title", "Break Over!",
		"-message", "Back to work!",
		"-sound", "Hero",
		"-group", "com.devlikebear.break-reminder",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notificationArgs() = %#v, want %#v", got, want)
	}

	for _, forbidden := range []string{"-activate", "-open", "-execute"} {
		for _, arg := range got {
			if arg == forbidden {
				t.Fatalf("notificationArgs() contains click action %q", forbidden)
			}
		}
	}
}

func TestNotificationArgsUseDefaultSound(t *testing.T) {
	got := notificationArgs("Title", "Message", "")
	want := []string{
		"-title", "Title",
		"-message", "Message",
		"-sound", "Glass",
		"-group", "com.devlikebear.break-reminder",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notificationArgs() = %#v, want %#v", got, want)
	}
}

func TestDarwinNotifierUsesTerminalNotifier(t *testing.T) {
	oldResolve := terminalNotifierPath
	oldRun := runCommand
	t.Cleanup(func() {
		terminalNotifierPath = oldResolve
		runCommand = oldRun
	})

	terminalNotifierPath = func() (string, error) {
		return "/opt/homebrew/bin/terminal-notifier", nil
	}
	var gotName string
	var gotArgs []string
	runCommand = func(name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := (&DarwinNotifier{}).Send("Break Over!", "Back to work!", "Hero"); err != nil {
		t.Fatal(err)
	}
	if gotName != "/opt/homebrew/bin/terminal-notifier" {
		t.Fatalf("command = %q, want terminal-notifier", gotName)
	}
	if !reflect.DeepEqual(gotArgs, notificationArgs("Break Over!", "Back to work!", "Hero")) {
		t.Fatalf("args = %#v", gotArgs)
	}
}

func TestDarwinNotifierWrapsCommandError(t *testing.T) {
	oldResolve := terminalNotifierPath
	oldRun := runCommand
	t.Cleanup(func() {
		terminalNotifierPath = oldResolve
		runCommand = oldRun
	})

	terminalNotifierPath = func() (string, error) {
		return "/opt/homebrew/bin/terminal-notifier", nil
	}
	runCommand = func(string, ...string) error { return errors.New("process failed") }

	err := (&DarwinNotifier{}).Send("Title", "Message", "Glass")
	if err == nil {
		t.Fatal("Send() error = nil, want command error")
	}
	if !strings.Contains(err.Error(), "send notification: process failed") {
		t.Fatalf("error = %q, want wrapped command error", err)
	}
}

func TestDarwinNotifierReturnsResolutionError(t *testing.T) {
	oldResolve := terminalNotifierPath
	oldRun := runCommand
	t.Cleanup(func() {
		terminalNotifierPath = oldResolve
		runCommand = oldRun
	})

	terminalNotifierPath = func() (string, error) {
		return "", errors.New("missing notifier")
	}
	runCommand = func(string, ...string) error {
		t.Fatal("runCommand called after path resolution failed")
		return nil
	}

	err := (&DarwinNotifier{}).Send("Title", "Message", "Glass")
	if err == nil || err.Error() != "missing notifier" {
		t.Fatalf("Send() error = %v, want resolution error", err)
	}
}
