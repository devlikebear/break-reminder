package config

// Default returns the default configuration matching the original bash script.
func Default() Config {
	return Config{
		WorkDurationMin:        50,
		BreakDurationMin:       10,
		IdleThresholdSec:       120,
		NaturalBreakSec:        300,
		WorkDays:               []int{1, 2, 3, 4, 5},
		WorkStartHour:          9,
		WorkStartMinute:        0,
		WorkEndHour:            18,
		WorkEndMinute:          0,
		Voice:                  "Yuna",
		TTSEngine:              "say",
		TTSModel:               "KittenML/kitten-tts-nano-0.8",
		TTSPythonCmd:           "python3",
		TTSAPIKey:              "",
		NotificationsEnabled:   true,
		TTSEnabled:             true,
		BreakActivitiesEnabled: true,
		AIEnabled:              false,
		AICLI:                  "claude",
		BreakScreenMode:        "ask",
		MaxLogLines:            1000,
		CheckIntervalSec:       60,
		Theme:                  "auto",
	}
}
