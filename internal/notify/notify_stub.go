//go:build !darwin

package notify

import "fmt"

type StubNotifier struct{}

func NewNotifier() Notifier {
	return &StubNotifier{}
}

func (n *StubNotifier) Send(title, message, sound string) error {
	fmt.Printf("[notification] %s: %s\n", title, message)
	return nil
}
