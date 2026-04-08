package state

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// State represents the application's current timer state.
type State struct {
	WorkSeconds            int    `json:"work_seconds"`
	Mode                   string `json:"mode"` // "work" or "break"
	LastCheck              int64  `json:"last_check"`
	BreakStart             int64  `json:"break_start"`
	TodayWorkSeconds       int    `json:"today_work_seconds"`
	TodayBreakSeconds      int    `json:"today_break_seconds"`
	LastUpdateDate         string `json:"last_update_date"`
	LastBreakWarningBucket int    `json:"last_break_warning_bucket"`
}

// DefaultStatePath returns ~/.break-reminder-state
func DefaultStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".break-reminder-state")
}

// New returns a fresh state.
func New() State {
	now := time.Now()
	return State{
		Mode:           "work",
		LastCheck:      now.Unix(),
		LastUpdateDate: now.Format("2006-01-02"),
	}
}

// Load reads state from the key=value file (compatible with bash version).
func Load(path string) (State, error) {
	s := New()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "WORK_SECONDS":
			if v, err := strconv.Atoi(value); err == nil {
				s.WorkSeconds = v
			}
		case "MODE":
			if value == "work" || value == "break" {
				s.Mode = value
			}
		case "LAST_CHECK":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				s.LastCheck = v
			}
		case "BREAK_START":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				s.BreakStart = v
			}
		case "TODAY_WORK_SECONDS":
			if v, err := strconv.Atoi(value); err == nil {
				s.TodayWorkSeconds = v
			}
		case "TODAY_BREAK_SECONDS":
			if v, err := strconv.Atoi(value); err == nil {
				s.TodayBreakSeconds = v
			}
		case "LAST_UPDATE_DATE":
			s.LastUpdateDate = value
		case "LAST_BREAK_WARNING_BUCKET":
			if v, err := strconv.Atoi(value); err == nil {
				s.LastBreakWarningBucket = v
			}
		}
	}

	return s, scanner.Err()
}

// Save writes state in key=value format (compatible with bash version).
func Save(path string, s State) error {
	content := fmt.Sprintf(`WORK_SECONDS=%d
MODE=%s
LAST_CHECK=%d
BREAK_START=%d
TODAY_WORK_SECONDS=%d
TODAY_BREAK_SECONDS=%d
LAST_UPDATE_DATE=%s
LAST_BREAK_WARNING_BUCKET=%d
`, s.WorkSeconds, s.Mode, s.LastCheck, s.BreakStart,
		s.TodayWorkSeconds, s.TodayBreakSeconds, s.LastUpdateDate, s.LastBreakWarningBucket)

	return os.WriteFile(path, []byte(content), 0o644)
}
