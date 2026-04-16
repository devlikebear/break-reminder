package doctor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/launchd"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/notify"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
	"github.com/devlikebear/break-reminder/internal/tts"
)

// Check represents a single diagnostic check.
type Check struct {
	Name   string
	Status string // "ok", "warn", "fail"
	Detail string
}

// Report contains all diagnostic results.
type Report struct {
	Checks []Check
}

func (r *Report) add(status, name, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: status, Detail: detail})
}

// FailCount returns the number of failed checks.
func (r *Report) FailCount() int {
	n := 0
	for _, c := range r.Checks {
		if c.Status == "fail" {
			n++
		}
	}
	return n
}

// Run performs all diagnostic checks.
func Run(cfg config.Config) Report {
	var r Report
	installHint := ttsInstallHint(cfg.TTSEngine)

	// Voice availability
	apiKey := tts.ResolveAPIKey(cfg)
	speaker := tts.NewSpeaker(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, apiKey)
	voiceLabel := cfg.TTSEngine + ":" + cfg.Voice
	if speaker.Available(cfg.Voice) {
		r.add("ok", "Voice ("+voiceLabel+")", "available")
	} else {
		detail := "not found"
		if installHint != "" {
			detail = "not found (" + installHint + ")"
		}
		r.add("fail", "Voice ("+voiceLabel+")", detail)
	}

	// TTS
	if err := tts.SpeakAndWait(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, apiKey, cfg.Voice, "테스트"); err != nil {
		detail := err.Error()
		if installHint != "" && !strings.Contains(detail, installHint) {
			detail += " (" + installHint + ")"
		}
		r.add("fail", "TTS", detail)
	} else {
		r.add("ok", "TTS", "working")
	}

	// Notification
	notifier := notify.NewNotifier()
	if err := notifier.Send("Break Reminder", "Doctor test", "Glass"); err != nil {
		r.add("fail", "Notification", err.Error())
	} else {
		r.add("ok", "Notification", "working")
	}

	// Idle detection
	detector := idle.NewDetector()
	idleSec := detector.IdleSeconds()
	r.add("ok", "Idle detection", fmt.Sprintf("current: %ds", idleSec))

	// State file
	statePath := state.DefaultStatePath()
	if _, err := os.Stat(statePath); err == nil {
		r.add("ok", "State file", statePath)
	} else {
		r.add("warn", "State file", "not found (will be created on first run)")
	}

	// Log file
	logPath := logging.DefaultLogPath()
	if info, err := os.Stat(logPath); err == nil {
		r.add("ok", "Log file", fmt.Sprintf("%s (%d bytes)", logPath, info.Size()))
	} else {
		r.add("warn", "Log file", "not found (will be created on first run)")
	}

	// LaunchAgent
	status := launchd.Status()
	switch {
	case status == "Not Installed":
		r.add("warn", "LaunchAgent", "not installed (run 'service install')")
	default:
		r.add("ok", "LaunchAgent", status)
	}

	menuBarStatus := launchd.MenuBarStatus()
	switch {
	case menuBarStatus == "Not Installed":
		r.add("info", "Menu bar auto-start", "not installed")
	default:
		r.add("ok", "Menu bar auto-start", menuBarStatus)
	}

	// Working hours
	if schedule.IsWorkingTime(cfg, time.Now()) {
		r.add("ok", "Working hours", "within working hours")
	} else {
		r.add("info", "Working hours", "outside working hours - inactive")
	}

	// Config file
	if _, err := os.Stat(config.ConfigPath()); err == nil {
		r.add("ok", "Config file", config.ConfigPath())
	} else {
		r.add("warn", "Config file", "not found (using defaults)")
	}

	return r
}

func ttsInstallHint(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "kitten", "kittentts":
		return "run 'break-reminder tts install kittentts'"
	case "supertonic":
		return "run 'break-reminder tts install supertonic'"
	case "gemini":
		return "set GEMINI_API_KEY env or tts_api_key in config"
	default:
		return ""
	}
}
