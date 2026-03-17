//go:build !darwin

package idle

// StubDetector always returns 0 on non-darwin platforms.
type StubDetector struct{}

func NewDetector() Detector {
	return &StubDetector{}
}

func (d *StubDetector) IdleSeconds() int {
	return 0
}
