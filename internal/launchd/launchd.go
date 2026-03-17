package launchd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Label = "com.devlikebear.break-reminder"
)

// PlistPath returns the LaunchAgent plist file path.
func PlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
}

// Install creates the plist and loads the agent.
func Install(binaryPath string) error {
	plist := generatePlist(binaryPath)
	path := PlistPath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	// Unload if previously loaded, ignore errors
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

// Uninstall unloads and removes the agent.
func Uninstall() error {
	path := PlistPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("agent not installed (plist not found)")
	}

	_ = exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}

// Start loads the agent.
func Start() error {
	path := PlistPath()
	_ = exec.Command("launchctl", "unload", path).Run()
	return exec.Command("launchctl", "load", path).Run()
}

// Stop unloads the agent.
func Stop() error {
	return exec.Command("launchctl", "unload", PlistPath()).Run()
}

// Status returns the current agent status.
func Status() string {
	path := PlistPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "Not Installed"
	}

	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return "Installed (status unknown)"
	}

	if strings.Contains(string(out), Label) {
		return "Installed & Running"
	}
	return "Installed (Not Loaded)"
}

func generatePlist(binaryPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>check</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/break-reminder.out</string>
    <key>StandardErrorPath</key>
    <string>/tmp/break-reminder.err</string>
</dict>
</plist>
`, Label, binaryPath)
}
