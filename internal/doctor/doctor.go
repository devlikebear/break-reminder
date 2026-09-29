package doctor

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

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
	"github.com/devlikebear/break-reminder/internal/timetools"
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
		r.add("ok", i18n.Text("Voice (")+voiceLabel+")", "available")
	} else {
		detail := i18n.Text("not found")
		if installHint != "" {
			detail = i18n.Text("not found (") + installHint + ")"
		}
		r.add("fail", i18n.Text("Voice (")+voiceLabel+")", detail)
	}

	// TTS
	if err := tts.SpeakAndWait(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, apiKey, cfg.Voice, i18n.Text("Test")); err != nil {
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
	if err := notifier.Send("Break Reminder", i18n.Text("Doctor test"), "Glass"); err != nil {
		r.add("fail", "Notification", err.Error()+i18n.Text("; timer completions remain in menu/dashboard; sound is not guaranteed"))
	} else {
		r.add("ok", "Notification", i18n.Text("request sent (banner visibility depends on macOS settings)"))
	}

	// Idle detection
	detector := idle.NewDetector()
	idleSec := detector.IdleSeconds()
	r.add("ok", i18n.Text("Idle detection"), fmt.Sprintf(i18n.Text("current: %ds"), idleSec))

	// State file
	statePath := state.DefaultStatePath()
	if _, err := os.Stat(statePath); err == nil {
		r.add("ok", i18n.Text("State file"), statePath)
	} else {
		r.add("warn", i18n.Text("State file"), i18n.Text("not found (will be created on first run)"))
	}

	// Log file
	logPath := logging.DefaultLogPath()
	if info, err := os.Stat(logPath); err == nil {
		r.add("ok", i18n.Text("Log file"), fmt.Sprintf(i18n.Text("%s (%d bytes)"), logPath, info.Size()))
	} else {
		r.add("warn", i18n.Text("Log file"), i18n.Text("not found (will be created on first run)"))
	}

	// LaunchAgent
	status := launchd.Status()
	switch {
	case status == "Not Installed":
		r.add("warn", "LaunchAgent", i18n.Text("not installed (run 'service install')"))
	default:
		r.add("ok", "LaunchAgent", status)
	}

	menuBarStatus := launchd.MenuBarStatus()
	switch {
	case menuBarStatus == "Not Installed":
		r.add("info", i18n.Text("Menu bar auto-start"), i18n.Text("not installed"))
	default:
		r.add("ok", i18n.Text("Menu bar auto-start"), menuBarStatus)
	}

	r.Checks = append(r.Checks, timeToolsDiagnostic(launchd.TimeToolsStatus(), timetools.ReadRuntime(timetools.NewStore(timetools.DefaultDirectory())), time.Now().UnixMilli()))

	// Work session
	now := time.Now()
	current, _ := state.Load(statePath)
	switch {
	case schedule.IsActive(cfg, current, now):
		r.add("ok", i18n.Text("Work session"), i18n.Text("running - timer active"))
	case current.SessionState == state.SessionStateEnded && current.SessionManual:
		r.add("info", i18n.Text("Work session"), i18n.Text("stopped by hand - run `break-reminder start` to resume"))
	case cfg.AutoSessionDetect && schedule.InDetectWindow(cfg, now):
		r.add("info", i18n.Text("Work session"), i18n.Text("not started - begins on your next activity"))
	case cfg.AutoSessionDetect:
		r.add("info", i18n.Text("Work session"), fmt.Sprintf(i18n.Text("outside the detection window (%02d:00-%02d:00 on working days)"),
			cfg.SessionDetectStartHour, cfg.SessionDetectEndHour))
	default:
		r.add("info", "Working hours", i18n.Text("outside working hours - inactive"))
	}

	// Timer mode
	if cfg.PomodoroEnabled() {
		r.add("ok", i18n.Text("Timer mode"), fmt.Sprintf(i18n.Text("pomodoro (%d/%d min, %d min long break every %d)"),
			cfg.PomodoroWorkMin, cfg.PomodoroBreakMin, cfg.PomodoroLongBreakMin, cfg.PomodoroLongBreakEvery))
	} else {
		r.add("ok", i18n.Text("Timer mode"), fmt.Sprintf(i18n.Text("classic (%d/%d min)"), cfg.WorkDurationMin, cfg.BreakDurationMin))
	}

	// Config file
	if _, err := os.Stat(config.ConfigPath()); err == nil {
		r.add("ok", i18n.Text("Config file"), config.ConfigPath())
	} else {
		r.add("warn", i18n.Text("Config file"), i18n.Text("not found (using defaults)"))
	}

	return r
}

func ttsInstallHint(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "kitten", "kittentts":
		return i18n.Text("run 'break-reminder tts install kittentts'")
	case "supertonic":
		return i18n.Text("run 'break-reminder tts install supertonic'")
	case "gemini":
		return i18n.Text("set GEMINI_API_KEY env or tts_api_key in config")
	default:
		return ""
	}
}
