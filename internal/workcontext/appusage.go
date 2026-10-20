package workcontext

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/devlikebear/break-reminder/internal/i18n"
)

// AppUsage is foreground time for one app. Only the app identity is kept:
// window titles and contents are never collected.
type AppUsage struct {
	BundleID string `json:"bundle_id"`
	Name     string `json:"name"`
	Seconds  int    `json:"seconds"`
}

// UsageStore keeps one JSON file per day under Dir.
type UsageStore struct{ Dir string }

var usageFile = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.json$`)

func (u UsageStore) path(day time.Time) string {
	return filepath.Join(u.Dir, day.Format("2006-01-02")+".json")
}

// Day returns the day's usage, most used first.
func (u UsageStore) Day(day time.Time) ([]AppUsage, error) {
	data, err := os.ReadFile(u.path(day))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []AppUsage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	sortUsage(items)
	return items, nil
}

// Add merges increments into the day's totals. The latest name wins so a
// renamed app shows its current name.
func (u UsageStore) Add(day time.Time, increments []AppUsage) error {
	for _, inc := range increments {
		if inc.BundleID == "" || inc.Seconds < 0 || inc.Seconds > 3600 {
			return fmt.Errorf(i18n.Text("invalid app usage increment for %q"), inc.BundleID)
		}
	}
	items, err := u.Day(day)
	if err != nil {
		return err
	}
	index := map[string]int{}
	for i, it := range items {
		index[it.BundleID] = i
	}
	for _, inc := range increments {
		if i, ok := index[inc.BundleID]; ok {
			items[i].Seconds += inc.Seconds
			if inc.Name != "" {
				items[i].Name = inc.Name
			}
			continue
		}
		index[inc.BundleID] = len(items)
		items = append(items, inc)
	}
	sortUsage(items)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return writeAtomic(u.path(day), data)
}

// Prune removes day files older than keepDays before today.
func (u UsageStore) Prune(today time.Time, keepDays int) error {
	y, m, d := today.Date()
	cutoff := time.Date(y, m, d, 0, 0, 0, 0, today.Location()).AddDate(0, 0, -(keepDays - 1))
	return u.each(func(name string, day time.Time) error {
		if day.Before(cutoff) {
			return os.Remove(filepath.Join(u.Dir, name))
		}
		return nil
	})
}

// DeleteAll removes every usage file (the "delete data" setting).
func (u UsageStore) DeleteAll() error {
	return u.each(func(name string, _ time.Time) error { return os.Remove(filepath.Join(u.Dir, name)) })
}

func (u UsageStore) each(fn func(name string, day time.Time) error) error {
	entries, err := os.ReadDir(u.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !usageFile.MatchString(e.Name()) {
			continue
		}
		day, err := time.ParseInLocation("2006-01-02.json", e.Name(), time.Local)
		if err != nil {
			continue
		}
		if err := fn(e.Name(), day); err != nil {
			return err
		}
	}
	return nil
}

// Top returns the n most used apps.
func Top(items []AppUsage, n int) []AppUsage {
	sorted := append([]AppUsage(nil), items...)
	sortUsage(sorted)
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

func sortUsage(items []AppUsage) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Seconds != items[j].Seconds {
			return items[i].Seconds > items[j].Seconds
		}
		return items[i].BundleID < items[j].BundleID
	})
}
