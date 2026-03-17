package idle

// Detector retrieves the system idle time.
type Detector interface {
	IdleSeconds() int
}
