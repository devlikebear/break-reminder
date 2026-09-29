//go:build darwin

package breakscreen

import (
	"fmt"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// askBreakMode shows an osascript dialog asking the user to choose between
// fullscreen blocking mode and notification-only mode.
// Returns "block" or "notify".
func askBreakMode() string {
	block := i18n.Text("Block Screen")
	notification := i18n.Text("Notification Only")
	message := i18n.Text("Break Time! How would you like to be reminded?") + "\n\n" + i18n.Text("• Block Screen: Full-screen overlay until break ends") + "\n" + i18n.Text("• Notification Only: Just show notifications")
	script := fmt.Sprintf(`display dialog %s buttons {%s, %s} default button %s with title %s with icon caution giving up after 30`, strconv.Quote(message), strconv.Quote(notification), strconv.Quote(block), strconv.Quote(block), strconv.Quote(i18n.Text("Break Reminder - Choose Mode")))

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Ask dialog failed, falling back to notify"))
		return "notify"
	}

	result := strings.TrimSpace(string(out))
	if strings.Contains(result, block) {
		return "block"
	}
	return "notify"
}
