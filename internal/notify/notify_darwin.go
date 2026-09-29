//go:build darwin

package notify

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"fmt"
	"os"
	"os/exec"
)

const notificationGroup = "com.devlikebear.break-reminder"

var (
	lookPath   = exec.LookPath
	fileExists = func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && !info.IsDir()
	}
	terminalNotifierPath = resolveTerminalNotifier
	runCommand           = func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}
)

type DarwinNotifier struct{}

func NewNotifier() Notifier {
	return &DarwinNotifier{}
}

func resolveTerminalNotifier() (string, error) {
	if path, err := lookPath("terminal-notifier"); err == nil {
		return path, nil
	}

	// LaunchAgents receive a minimal PATH, so probe both Homebrew prefixes.
	for _, path := range []string{
		"/opt/homebrew/bin/terminal-notifier",
		"/usr/local/bin/terminal-notifier",
	} {
		if fileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("%s", i18n.Text("terminal-notifier not found; install it with 'brew install terminal-notifier'"))
}

func notificationArgs(title, message, sound string) []string {
	if sound == "" {
		sound = "Glass"
	}
	return []string{
		"-title", title,
		"-message", message,
		"-sound", sound,
		"-group", notificationGroup,
	}
}

func (n *DarwinNotifier) Send(title, message, sound string) error {
	path, err := terminalNotifierPath()
	if err != nil {
		return err
	}
	if err := runCommand(path, notificationArgs(title, message, sound)...); err != nil {
		return fmt.Errorf(i18n.Text("send notification: %w"), err)
	}
	return nil
}

// Available checks the backend without sending an unsolicited notification.
func Available() bool { _, err := terminalNotifierPath(); return err == nil }

var runEventCommand = func(ctx context.Context, path string, args ...string) error {
	return exec.CommandContext(ctx, path, args...).Run()
}

// SendEvent uses its own group so work/break notifications cannot replace it.
func SendEvent(ctx context.Context, title, message, eventID string) error {
	path, err := terminalNotifierPath()
	if err != nil {
		return err
	}
	args := notificationArgs(title, message, "Glass")
	args[len(args)-1] = notificationGroup + ".time-tools." + eventID
	return runEventCommand(ctx, path, args...)
}
