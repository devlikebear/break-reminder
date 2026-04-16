package config

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
}

// WorkDurationSec returns work duration in seconds.
func (c Config) WorkDurationSec() int {
	return c.WorkDurationMin * 60
}

// BreakDurationSec returns break duration in seconds.
func (c Config) BreakDurationSec() int {
	return c.BreakDurationMin * 60
}
