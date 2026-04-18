package state

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	ErrBreakNotActive = errors.New("snooze requires an active break")
	ErrStatePaused    = errors.New("cannot snooze while paused")
	ErrInvalidSnooze  = errors.New("snooze duration must be greater than zero")
)

// State represents the application's current timer state.
type State struct {
	WorkSeconds            int    `json:"work_seconds"`
	Mode                   string `json:"mode"` // "work" or "break"
	LastCheck              int64  `json:"last_check"`
	BreakStart             int64  `json:"break_start"`
	SnoozeUntil            int64  `json:"snooze_until"`
	Paused                 bool   `json:"paused"`
	PausedAt               int64  `json:"paused_at"`
	TodayWorkSeconds       int    `json:"today_work_seconds"`
	TodayBreakSeconds      int    `json:"today_break_seconds"`
	LastUpdateDate         string `json:"last_update_date"`
	LastBreakWarningBucket int    `json:"last_break_warning_bucket"`
	HourlyWork             [24]int `json:"hourly_work"`
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

// EnterBreak updates the state for a fresh break transition.
func (s State) EnterBreak(at int64) State {
	s.Mode = "break"
	s.BreakStart = at
	s.SnoozeUntil = 0
	s.WorkSeconds = 0
	s.Paused = false
	s.PausedAt = 0
	s.LastBreakWarningBucket = 0
	return s
}

// Pause freezes the timer until Resume is called.
func (s State) Pause(at int64) State {
	if s.Paused {
		return s
	}
	s = s.accrueUntil(at)
	s.Paused = true
	s.PausedAt = at
	return s
}

func (s State) accrueUntil(at int64) State {
	if at <= 0 {
		return s
	}
	if s.LastUpdateDate == "" {
		s.LastUpdateDate = time.Unix(at, 0).In(time.Local).Format("2006-01-02")
	}
	if s.LastCheck <= 0 || at <= s.LastCheck {
		if s.LastCheck <= 0 {
			s.LastCheck = at
		}
		return s
	}

	cursor := time.Unix(s.LastCheck, 0).In(time.Local)
	target := time.Unix(at, 0).In(time.Local)
	currentDate := cursor.Format("2006-01-02")
	if s.LastUpdateDate == "" {
		s.LastUpdateDate = currentDate
	}

	for currentDate != target.Format("2006-01-02") {
		nextMidnight := time.Date(cursor.Year(), cursor.Month(), cursor.Day()+1, 0, 0, 0, 0, time.Local)
		s = s.addElapsed(int(nextMidnight.Unix() - cursor.Unix()))
		s.TodayWorkSeconds = 0
		s.TodayBreakSeconds = 0
		cursor = nextMidnight
		currentDate = cursor.Format("2006-01-02")
		s.LastUpdateDate = currentDate
	}

	s = s.addElapsed(int(target.Unix() - cursor.Unix()))
	s.LastCheck = at
	if s.LastUpdateDate == "" {
		s.LastUpdateDate = target.Format("2006-01-02")
	}
	return s
}

func (s State) addElapsed(elapsed int) State {
	if elapsed <= 0 {
		return s
	}
	switch s.Mode {
	case "break":
		s.TodayBreakSeconds += elapsed
	default:
		s.WorkSeconds += elapsed
		s.TodayWorkSeconds += elapsed
	}
	return s
}

// Resume unfreezes the timer and shifts time anchors so paused time is not counted.
func (s State) Resume(at int64) State {
	if !s.Paused {
		return s
	}
	if s.PausedAt <= 0 {
		if s.LastCheck > 0 {
			s.LastCheck = at
		}
		if s.Mode == "break" && s.BreakStart > 0 {
			s.BreakStart = at
		}
		s.Paused = false
		s.PausedAt = 0
		return s
	}
	gap := at - s.PausedAt
	if gap < 0 {
		gap = 0
	}
	if s.LastCheck > 0 {
		s.LastCheck += gap
	}
	if s.Mode == "break" && s.BreakStart > 0 {
		s.BreakStart += gap
	}
	if s.SnoozeUntil > 0 {
		s.SnoozeUntil += gap
	}
	s.Paused = false
	s.PausedAt = 0
	return s
}

// SnoozeBreak ends the current break and postpones the next break until now+d.
func (s State) SnoozeBreak(now time.Time, d time.Duration) (State, error) {
	if d <= 0 {
		return s, ErrInvalidSnooze
	}
	if s.Paused || s.Mode == "paused" {
		return s, ErrStatePaused
	}
	if s.Mode != "break" {
		return s, ErrBreakNotActive
	}

	nowUnix := now.Unix()
	today := now.Format("2006-01-02")
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	if s.LastUpdateDate != today {
		s.TodayWorkSeconds = 0
		s.TodayBreakSeconds = 0
		s.LastUpdateDate = today
	} else if s.LastUpdateDate == "" {
		s.LastUpdateDate = today
	}

	if s.LastCheck > 0 {
		elapsedStart := s.LastCheck
		if elapsedStart < startOfDay {
			elapsedStart = startOfDay
		}
		if elapsed := int(nowUnix - elapsedStart); elapsed > 0 {
			s.TodayBreakSeconds += elapsed
		}
	}

	s.Mode = "work"
	s.LastCheck = nowUnix
	s.BreakStart = 0
	s.SnoozeUntil = now.Add(d).Unix()
	s.WorkSeconds = 0
	s.LastBreakWarningBucket = 0
	return s, nil
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
			if value == "work" || value == "break" || value == "paused" {
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
		case "SNOOZE_UNTIL":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				s.SnoozeUntil = v
			}
		case "PAUSED":
			s.Paused = value == "true"
		case "PAUSED_AT":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				s.PausedAt = v
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
		case "HOURLY_WORK":
			parts := strings.Split(value, ",")
			if len(parts) == 24 {
				for i, p := range parts {
					if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
						s.HourlyWork[i] = n
					}
				}
			}
		}
	}

	return s, scanner.Err()
}

func serialize(s State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "WORK_SECONDS=%d\n", s.WorkSeconds)
	fmt.Fprintf(&b, "MODE=%s\n", s.Mode)
	fmt.Fprintf(&b, "LAST_CHECK=%d\n", s.LastCheck)
	fmt.Fprintf(&b, "BREAK_START=%d\n", s.BreakStart)
	fmt.Fprintf(&b, "SNOOZE_UNTIL=%d\n", s.SnoozeUntil)
	fmt.Fprintf(&b, "PAUSED=%t\n", s.Paused)
	fmt.Fprintf(&b, "PAUSED_AT=%d\n", s.PausedAt)
	fmt.Fprintf(&b, "TODAY_WORK_SECONDS=%d\n", s.TodayWorkSeconds)
	fmt.Fprintf(&b, "TODAY_BREAK_SECONDS=%d\n", s.TodayBreakSeconds)
	fmt.Fprintf(&b, "LAST_UPDATE_DATE=%s\n", s.LastUpdateDate)
	fmt.Fprintf(&b, "LAST_BREAK_WARNING_BUCKET=%d\n", s.LastBreakWarningBucket)

	hourlyParts := make([]string, 24)
	for i, v := range s.HourlyWork {
		hourlyParts[i] = strconv.Itoa(v)
	}
	fmt.Fprintf(&b, "HOURLY_WORK=%s\n", strings.Join(hourlyParts, ","))

	return b.String()
}

func saveUnlocked(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(serialize(s)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func withStateLock(path string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	lockFile, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	return fn()
}

// Update loads, mutates, and saves state under an exclusive file lock.
func Update(path string, updateFn func(State) (State, error)) error {
	return withStateLock(path, func() error {
		s, err := Load(path)
		if err != nil {
			return err
		}
		updated, err := updateFn(s)
		if err != nil {
			return err
		}
		return saveUnlocked(path, updated)
	})
}

// Save writes state in key=value format (compatible with bash version).
func Save(path string, s State) error {
	return withStateLock(path, func() error {
		return saveUnlocked(path, s)
	})
}
