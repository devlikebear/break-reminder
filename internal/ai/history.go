package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DailySummary represents one day's usage statistics.
type DailySummary struct {
	Date       string `json:"date"`
	WorkMin    int    `json:"work_min"`
	BreakMin   int    `json:"break_min"`
	Sessions   int    `json:"sessions"`
	Activities int    `json:"activities"`
}

// historyPathOverride allows tests to redirect the history file.
var historyPathOverride string

// HistoryPath returns ~/.break-reminder-history.json
func HistoryPath() string {
	if historyPathOverride != "" {
		return historyPathOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".break-reminder-history.json")
}

// LoadHistory reads the history file.
func LoadHistory() ([]DailySummary, error) {
	data, err := os.ReadFile(HistoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var history []DailySummary
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// AppendHistory adds a daily summary to the history file.
func AppendHistory(summary DailySummary) error {
	history, err := LoadHistory()
	if err != nil {
		history = nil
	}

	// Update existing entry for same date or append
	found := false
	for i, h := range history {
		if h.Date == summary.Date {
			history[i] = summary
			found = true
			break
		}
	}
	if !found {
		history = append(history, summary)
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(HistoryPath(), data, 0o644)
}
