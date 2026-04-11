package launchd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Label        = "com.devlikebear.break-reminder"
	MenuBarLabel = Label + ".menubar"
)

var userHomeDir = os.UserHomeDir

// PlistPath returns the LaunchAgent plist file path.
func PlistPath() string {
	return plistPath(Label)
}

// MenuBarPlistPath returns the LaunchAgent plist file path for the menu bar app.
func MenuBarPlistPath() string {
	return plistPath(MenuBarLabel)
}

func plistPath(label string) string {
	home, _ := userHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

func writePlist(path, plist string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	return nil
}

func loadJob(path string) error {
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}
	return nil
}

func unloadJob(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := exec.Command("launchctl", "unload", path).Run(); err != nil {
		return fmt.Errorf("launchctl unload: %w", err)
	}
	return nil
}

func removeJob(path string) error {
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Install creates the plists and loads the agents.
func Install(binaryPath, menuBarPath string) (bool, error) {
	if err := writePlist(PlistPath(), generateTimerPlist(binaryPath)); err != nil {
		return false, err
	}
	if err := loadJob(PlistPath()); err != nil {
		return false, err
	}

	if menuBarPath == "" {
		if err := removeJob(MenuBarPlistPath()); err != nil {
			return false, fmt.Errorf("remove stale menu bar agent: %w", err)
		}
		return false, nil
	}

	if err := writePlist(MenuBarPlistPath(), generateMenuBarPlist(menuBarPath)); err != nil {
		return false, err
	}
	if err := loadJob(MenuBarPlistPath()); err != nil {
		return false, err
	}

	return true, nil
}

// Uninstall unloads and removes the agents.
func Uninstall() error {
	timerInstalled := Status() != "Not Installed"
	menuInstalled := MenuBarStatus() != "Not Installed"
	if !timerInstalled && !menuInstalled {
		return fmt.Errorf("agent not installed (plist not found)")
	}

	var errs []error
	if timerInstalled {
		if err := removeJob(PlistPath()); err != nil {
			errs = append(errs, fmt.Errorf("remove timer agent: %w", err))
		}
	}
	if menuInstalled {
		if err := removeJob(MenuBarPlistPath()); err != nil {
			errs = append(errs, fmt.Errorf("remove menu bar agent: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Start loads the agents.
func Start() error {
	var errs []error
	if err := loadJob(PlistPath()); err != nil {
		errs = append(errs, fmt.Errorf("start timer agent: %w", err))
	}
	if MenuBarStatus() != "Not Installed" {
		if err := loadJob(MenuBarPlistPath()); err != nil {
			errs = append(errs, fmt.Errorf("start menu bar agent: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Stop unloads the agents.
func Stop() error {
	var errs []error
	if err := unloadJob(PlistPath()); err != nil {
		errs = append(errs, fmt.Errorf("stop timer agent: %w", err))
	}
	if MenuBarStatus() != "Not Installed" {
		if err := unloadJob(MenuBarPlistPath()); err != nil {
			errs = append(errs, fmt.Errorf("stop menu bar agent: %w", err))
		}
	}
	return errors.Join(errs...)
}

func jobStatus(path, label string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "Not Installed"
	}

	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return "Installed (status unknown)"
	}

	if strings.Contains(string(out), label) {
		return "Installed & Running"
	}
	return "Installed (Not Loaded)"
}

// Status returns the current timer agent status.
func Status() string {
	return jobStatus(PlistPath(), Label)
}

// MenuBarStatus returns the current menu bar agent status.
func MenuBarStatus() string {
	return jobStatus(MenuBarPlistPath(), MenuBarLabel)
}

func generateTimerPlist(binaryPath string) string {
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

func generateMenuBarPlist(menuBarPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>LimitLoadToSessionType</key>
    <string>Aqua</string>
    <key>StandardOutPath</key>
    <string>/tmp/break-reminder-menubar.out</string>
    <key>StandardErrorPath</key>
    <string>/tmp/break-reminder-menubar.err</string>
</dict>
</plist>
`, MenuBarLabel, menuBarPath)
}
