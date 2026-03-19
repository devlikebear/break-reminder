//go:build darwin

package breakscreen

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// showOverlay launches the Swift break-screen helper as a subprocess.
// It blocks until the helper exits (timer complete or user skipped).
func showOverlay(breakDurSec int, breakStartUnix int64, todayWorkMin, todayBreakMin int) {
	helperPath := FindHelper("break-screen")
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
		"--work-min", strconv.Itoa(todayWorkMin),
		"--break-min", strconv.Itoa(todayBreakMin),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Str("helper", helperPath).Int("remaining_sec", remaining).Msg("Launching break screen overlay")

	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Msg("Break screen helper exited with error")
	}
}

