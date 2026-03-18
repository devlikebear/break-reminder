//go:build darwin

package breakscreen

import (
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// askBreakMode shows an osascript dialog asking the user to choose between
// fullscreen blocking mode and notification-only mode.
// Returns "block" or "notify".
func askBreakMode() string {
	script := `display dialog "Break Time! How would you like to be reminded?" & return & return & ` +
		`"• Block Screen: Full-screen overlay until break ends" & return & ` +
		`"• Notification Only: Just show notifications" ` +
		`buttons {"Notification Only", "Block Screen"} ` +
		`default button "Block Screen" ` +
		`with title "Break Reminder - Choose Mode" ` +
		`with icon caution ` +
		`giving up after 30`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		log.Warn().Err(err).Msg("Ask dialog failed, falling back to notify")
		return "notify"
	}

	result := strings.TrimSpace(string(out))
	if strings.Contains(result, "Block Screen") {
		return "block"
	}
	return "notify"
}
