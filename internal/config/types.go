package config

// Timer modes selectable through the timer_mode config key.
const (
	TimerModeClassic  = "classic"
	TimerModePomodoro = "pomodoro"
)

// Config holds all application configuration.
type Config struct {
	WorkDurationMin        int    `yaml:"work_duration_min"`
	BreakDurationMin       int    `yaml:"break_duration_min"`
	IdleThresholdSec       int    `yaml:"idle_threshold_sec"`
	NaturalBreakSec        int    `yaml:"natural_break_sec"`
	WorkDays               []int  `yaml:"work_days"`
	WorkStartHour          int    `yaml:"work_start_hour"`
	WorkStartMinute        int    `yaml:"work_start_minute"`
	WorkEndHour            int    `yaml:"work_end_hour"`
	WorkEndMinute          int    `yaml:"work_end_minute"`
	AutoSessionDetect      bool   `yaml:"auto_session_detect"`
	SessionDetectStartHour int    `yaml:"session_detect_start_hour"`
	SessionDetectEndHour   int    `yaml:"session_detect_end_hour"`
	SessionIdleEndMin      int    `yaml:"session_idle_end_min"`
	TimerMode              string `yaml:"timer_mode"` // "classic", "pomodoro"
	PomodoroWorkMin        int    `yaml:"pomodoro_work_min"`
	PomodoroBreakMin       int    `yaml:"pomodoro_break_min"`
	PomodoroLongBreakMin   int    `yaml:"pomodoro_long_break_min"`
	PomodoroLongBreakEvery int    `yaml:"pomodoro_long_break_every"`
	Voice                  string `yaml:"voice"`
	TTSEngine              string `yaml:"tts_engine"`
	TTSModel               string `yaml:"tts_model"`
	TTSPythonCmd           string `yaml:"tts_python_cmd"`
	TTSAPIKey              string `yaml:"tts_api_key"`
	NotificationsEnabled   bool   `yaml:"notifications_enabled"`
	TTSEnabled             bool   `yaml:"tts_enabled"`
	BreakActivitiesEnabled bool   `yaml:"break_activities_enabled"`
	AIEnabled              bool   `yaml:"ai_enabled"`
	AICLI                  string `yaml:"ai_cli"`
	BreakScreenMode        string `yaml:"break_screen_mode"` // "ask", "block", "notify"
	MaxLogLines            int    `yaml:"max_log_lines"`
	CheckIntervalSec       int    `yaml:"check_interval_sec"`
	Theme                  string `yaml:"theme"` // "auto", "dark", "light"
}

// WorkDurationSec returns the configured classic work duration in seconds.
func (c Config) WorkDurationSec() int {
	return c.WorkDurationMin * 60
}

// BreakDurationSec returns the configured classic break duration in seconds.
func (c Config) BreakDurationSec() int {
	return c.BreakDurationMin * 60
}

// PomodoroEnabled reports whether the pomodoro timer mode is active.
func (c Config) PomodoroEnabled() bool {
	return c.TimerMode == TimerModePomodoro
}

// EffectiveWorkMin returns the work duration of the current timer mode.
func (c Config) EffectiveWorkMin() int {
	if c.PomodoroEnabled() {
		return c.PomodoroWorkMin
	}
	return c.WorkDurationMin
}

// EffectiveWorkSec returns the work duration of the current timer mode in seconds.
func (c Config) EffectiveWorkSec() int {
	return c.EffectiveWorkMin() * 60
}

// LongBreakDue reports whether the break following the completed-th pomodoro
// of the current cycle is a long break. It is always false outside pomodoro mode.
func (c Config) LongBreakDue(completed int) bool {
	if !c.PomodoroEnabled() {
		return false
	}
	if c.PomodoroLongBreakEvery <= 0 || completed <= 0 {
		return false
	}
	return completed%c.PomodoroLongBreakEvery == 0
}

// EffectiveBreakMin returns the break duration for the given number of
// completed pomodoros in the current cycle.
func (c Config) EffectiveBreakMin(completed int) int {
	if !c.PomodoroEnabled() {
		return c.BreakDurationMin
	}
	if c.LongBreakDue(completed) {
		return c.PomodoroLongBreakMin
	}
	return c.PomodoroBreakMin
}

// EffectiveBreakSec returns the break duration in seconds for the given number
// of completed pomodoros in the current cycle.
func (c Config) EffectiveBreakSec(completed int) int {
	return c.EffectiveBreakMin(completed) * 60
}

// SessionIdleEndSec returns the idle span that ends a work session, in seconds.
func (c Config) SessionIdleEndSec() int {
	return c.SessionIdleEndMin * 60
}
