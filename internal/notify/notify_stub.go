//go:build !darwin

package notify

import (
	"context"
	"fmt"
)

type StubNotifier struct{}

func NewNotifier() Notifier {
	return &StubNotifier{}
}

func (n *StubNotifier) Send(title, message, sound string) error {
	fmt.Printf("[notification] %s: %s\n", title, message)
	return nil
}

func Available() bool { return false }
func SendEvent(ctx context.Context, title, message, eventID string) error {
	return fmt.Errorf("desktop notifications require macOS")
}
