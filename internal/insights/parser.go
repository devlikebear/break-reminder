package insights

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"encoding/json"
	"fmt"
	"strings"
)

// ParseResponse parses the AI response into a Report (without GeneratedAt).
// Strips markdown code fences if present.
func ParseResponse(raw string) (*Report, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var r Report
	if err := json.Unmarshal([]byte(cleaned), &r); err != nil {
		return nil, fmt.Errorf(i18n.Text("parse AI response: %w"), err)
	}
	return &r, nil
}
