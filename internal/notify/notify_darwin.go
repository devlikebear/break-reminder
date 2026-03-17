//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

type DarwinNotifier struct{}

func NewNotifier() Notifier {
	return &DarwinNotifier{}
}

func (n *DarwinNotifier) Send(title, message, sound string) error {
	if sound == "" {
		sound = "Glass"
	}
	script := fmt.Sprintf(`display notification %q with title %q sound name %q`, message, title, sound)
	return exec.Command("osascript", "-e", script).Run()
}
