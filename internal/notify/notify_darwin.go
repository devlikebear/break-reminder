//go:build darwin

package notify

import (
	"fmt"
	"os"
	"os/exec"
)

const notificationGroup = "com.devlikebear.break-reminder"

var (
	lookPath = exec.LookPath
	fileExists = func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && !info.IsDir()
	}
	terminalNotifierPath = resolveTerminalNotifier
	runCommand = func(name string, args ...string) error {
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

	return "", fmt.Errorf("terminal-notifier not found; install it with 'brew install terminal-notifier'")
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
		return fmt.Errorf("send notification: %w", err)
	}
	return nil
}
