//go:build !darwin

package breakscreen

import "fmt"

func showOverlay(workMin, breakDurSec int, breakStartUnix int64, todayWorkMin, todayBreakMin int) {
	fmt.Println("[break-screen] Fullscreen overlay is only supported on macOS")
	sendNotification(workMin, breakDurSec/60)
}

func askBreakMode() string {
	return "notify"
}
