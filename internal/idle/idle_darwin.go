//go:build darwin

package idle

import (
	"os/exec"
	"strconv"
	"strings"
)

// DarwinDetector reads idle time from IOHIDSystem on macOS.
type DarwinDetector struct{}

func NewDetector() Detector {
	return &DarwinDetector{}
}

func (d *DarwinDetector) IdleSeconds() int {
	out, err := exec.Command("/usr/sbin/ioreg", "-c", "IOHIDSystem").Output()
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "HIDIdleTime") {
			parts := strings.Fields(line)
			if len(parts) == 0 {
				continue
			}
			val := parts[len(parts)-1]
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				continue
			}
			return int(n / 1_000_000_000) // nanoseconds to seconds
		}
	}
	return 0
}
