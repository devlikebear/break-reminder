package insights

import (
	"context"
	"time"

	"github.com/devlikebear/break-reminder/internal/ai"
)

// AIClient is the minimal interface Generate needs.
// The real ai.Client satisfies this automatically.
type AIClient interface {
	Query(ctx context.Context, prompt string) (string, error)
}

// Generate orchestrates building the prompt, calling the AI, and assembling the Report.
// It does NOT save the result — caller should call Save() if persistence is desired.
func Generate(ctx context.Context, client AIClient, history []ai.DailySummary, now time.Time) (*Report, error) {
	prompt := BuildPrompt(history)
	raw, err := client.Query(ctx, prompt)
	if err != nil {
		return nil, err
	}
	report, err := ParseResponse(raw)
	if err != nil {
		return nil, err
	}
	report.GeneratedAt = now.Format(time.RFC3339)
	return report, nil
}
