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
		WorkEndHour:            18,
		Voice:                  "Yuna",
		NotificationsEnabled:   true,
		TTSEnabled:             true,
		BreakActivitiesEnabled: true,
		AIEnabled:              false,
		AICLI:                  "claude",
		MaxLogLines:            1000,
		CheckIntervalSec:       60,
	}
}
