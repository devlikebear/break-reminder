package insights

import (
	"encoding/json"
	"fmt"
	"github.com/devlikebear/break-reminder/internal/i18n"

	"github.com/devlikebear/break-reminder/internal/ai"
)

const promptTemplate = `Analyze the user's recent work/break history:

%s

Return JSON only, with no markdown fences:
- daily_report: a concise 2-3 sentence summary
- patterns: 2-3 objects with type, title, description, suggestion
- type must be "warning" for concerns such as insufficient breaks or slumps, "positive" for improvements, or "info" for neutral observations such as optimal working hours
Write daily_report, title, description, and suggestion in %s.
Preserve the JSON field names and type values exactly.

{"daily_report":"...","patterns":[{"type":"info","title":"...","description":"...","suggestion":"..."}]}`

// BuildPrompt constructs the AI prompt from the given history entries.
func BuildPrompt(history []ai.DailySummary) string {
	if history == nil {
		history = []ai.DailySummary{}
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		data = []byte("[]")
	}
	return fmt.Sprintf(promptTemplate, string(data), i18n.ResponseLanguage())
}
