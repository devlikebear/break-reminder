//go:build darwin

package breakscreen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// showOverlay launches the Swift break-screen helper as a subprocess.
// It blocks until the helper exits (timer complete or user skipped).
func showOverlay(breakDurSec int, breakStartUnix int64) {
	helperPath := findHelper()
	if helperPath == "" {
		log.Warn().Msg("break-screen helper not found, falling back to notification")
		sendNotification()
		return
	}

	// Calculate remaining break time
	elapsed := int(time.Now().Unix() - breakStartUnix)
	remaining := breakDurSec - elapsed
	if remaining <= 0 {
		return
	}

	cmd := exec.Command(helperPath,
		"--duration", strconv.Itoa(remaining),
		"--skip-after", "120",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Str("helper", helperPath).Int("remaining_sec", remaining).Msg("Launching break screen overlay")

	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Msg("Break screen helper exited with error")
	}
}

// findHelper searches for the break-screen binary in common locations.
func findHelper() string {
	// 1. Next to the main binary
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "break-screen")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 2. In the project's bin/ directory (development)
	candidates := []string{
		"bin/break-screen",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "break-screen"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	// 3. In PATH
	if p, err := exec.LookPath("break-screen"); err == nil {
		return p
	}

	fmt.Fprintln(os.Stderr, "break-screen helper not found. Run 'make build-helper' to build it.")
	return ""
}
