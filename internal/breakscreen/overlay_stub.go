//go:build !darwin

package breakscreen

import "github.com/devlikebear/break-reminder/internal/i18n"

import "fmt"

func showOverlay(workMin, breakDurSec int, breakStartUnix int64, todayWorkMin, todayBreakMin int) {
	fmt.Println(i18n.Text("[break-screen] Fullscreen overlay is only supported on macOS"))
	sendNotification(workMin, breakDurSec/60)
}

func askBreakMode() string {
	return "notify"
}
