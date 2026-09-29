package launchd

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
)

const TimeToolsLabel = Label + ".timetools"

var removeRuntimeJob = removeJob

func TimeToolsPlistPath() string { return plistPath(TimeToolsLabel) }
func TimeToolsStatus() string    { return jobStatus(TimeToolsPlistPath(), TimeToolsLabel) }
func generateTimeToolsPlist(binary string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>time-tools</string><string>run</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><true/>
<key>LimitLoadToSessionType</key><string>Aqua</string>
<key>ThrottleInterval</key><integer>10</integer>
<key>StandardOutPath</key><string>%s</string>
<key>StandardErrorPath</key><string>%s</string>
</dict></plist>`, TimeToolsLabel, html.EscapeString(binary), html.EscapeString(filepath.Join(updaterLogDir(), "break-reminder-timetools.out")), html.EscapeString(filepath.Join(updaterLogDir(), "break-reminder-timetools.err")))
}
func installTimeTools(binary string, load bool) error {
	if err := os.MkdirAll(updaterLogDir(), 0700); err != nil {
		return err
	}
	if err := writePlist(TimeToolsPlistPath(), generateTimeToolsPlist(binary)); err != nil {
		return err
	}
	if load {
		return loadJobForInstall(TimeToolsPlistPath())
	}
	return nil
}

type JobInstallation struct {
	Installed bool `json:"installed"`
	Loaded    bool `json:"loaded"`
}
type RuntimeInstallation struct {
	Timer     JobInstallation `json:"timer"`
	MenuBar   JobInstallation `json:"menubar"`
	TimeTools JobInstallation `json:"time_tools"`
}

func CaptureRuntimeInstallation() RuntimeInstallation {
	capture := func(path, label string) JobInstallation {
		_, err := os.Stat(path)
		installed := err == nil
		return JobInstallation{Installed: installed, Loaded: installed && exec.Command("launchctl", "list", label).Run() == nil}
	}
	return RuntimeInstallation{Timer: capture(PlistPath(), Label), MenuBar: capture(MenuBarPlistPath(), MenuBarLabel), TimeTools: capture(TimeToolsPlistPath(), TimeToolsLabel)}
}

// MigrateRuntime is run by the newly installed binary. Never reload its updater.
func MigrateRuntime(binary, menu string, prior RuntimeInstallation) error {
	var errs []error
	if prior.Timer.Installed {
		if err := writePlist(PlistPath(), generateTimerPlist(binary)); err != nil {
			errs = append(errs, err)
		} else if prior.Timer.Loaded {
			if err = loadJobForInstall(PlistPath()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if prior.MenuBar.Installed {
		if menu == "" {
			errs = append(errs, fmt.Errorf("installed menu bar helper is missing"))
		} else if err := writePlist(MenuBarPlistPath(), generateMenuBarPlist(menu)); err != nil {
			errs = append(errs, err)
		} else if prior.MenuBar.Loaded {
			if err = loadJobForInstall(MenuBarPlistPath()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	// Only existing managed installations qualify for adding a missing worker.
	if prior.TimeTools.Installed || (prior.Timer.Installed && prior.Timer.Loaded) {
		load := prior.TimeTools.Loaded || (!prior.TimeTools.Installed && prior.Timer.Loaded)
		if err := installTimeTools(binary, load); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if prior.Timer.Installed || prior.TimeTools.Installed {
		data, _ := json.Marshal(map[string]int{"runtime_schema": 1})
		if err := os.WriteFile(filepath.Join(filepath.Dir(PlistPath()), ".break-reminder-runtime-version"), data, 0600); err != nil {
			return err
		}
	}
	return nil
}
