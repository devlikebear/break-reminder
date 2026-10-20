package insights

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/devlikebear/break-reminder/internal/paths"
)

// Pattern represents a single AI-discovered pattern insight.
type Pattern struct {
	Type        string `json:"type"` // "warning", "positive", "info"
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// Report is the top-level insights document.
type Report struct {
	GeneratedAt string    `json:"generated_at"`
	DailyReport string    `json:"daily_report"`
	Patterns    []Pattern `json:"patterns"`
}

// pathOverride lets tests redirect the insights file.
var pathOverride string

// Path returns the insights file (see paths.InsightsFile).
func Path() string {
	if pathOverride != "" {
		return pathOverride
	}
	return paths.InsightsFile()
}

// Load reads the insights file. Returns (nil, nil) if the file doesn't exist.
func Load() (*Report, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Save writes the report atomically, creating parent directories if needed.
func Save(r *Report) error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o644)
}
