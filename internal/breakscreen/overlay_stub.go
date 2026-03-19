//go:build !darwin

package breakscreen

import "fmt"

func showOverlay(breakDurSec int, breakStartUnix int64, todayWorkMin, todayBreakMin int) {
	fmt.Println("[break-screen] Fullscreen overlay is only supported on macOS")
	sendNotification()
}

func askBreakMode() string {
	return "notify"
}
