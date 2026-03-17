package notify

// Notifier sends desktop notifications.
type Notifier interface {
	Send(title, message, sound string) error
}
