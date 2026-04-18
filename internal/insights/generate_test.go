package insights

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/ai"
)

type fakeAIClient struct {
	response string
	err      error
}

func (f *fakeAIClient) Query(ctx context.Context, prompt string) (string, error) {
	return f.response, f.err
}

func TestGenerateSuccess(t *testing.T) {
	client := &fakeAIClient{
		response: `{"daily_report":"test report","patterns":[{"type":"info","title":"T","description":"D","suggestion":"S"}]}`,
	}
	history := []ai.DailySummary{{Date: "2026-04-17", WorkMin: 60}}

	report, err := Generate(context.Background(), client, history, time.Now())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if report.DailyReport != "test report" {
		t.Errorf("DailyReport mismatch: %q", report.DailyReport)
	}
	if report.GeneratedAt == "" {
		t.Error("GeneratedAt should be set")
	}
}

func TestGenerateAIError(t *testing.T) {
	client := &fakeAIClient{err: errors.New("CLI not found")}
	_, err := Generate(context.Background(), client, nil, time.Now())
	if err == nil {
		t.Error("expected error from failed AI call")
	}
}

func TestGenerateInvalidResponse(t *testing.T) {
	client := &fakeAIClient{response: "not json"}
	_, err := Generate(context.Background(), client, nil, time.Now())
	if err == nil {
		t.Error("expected error from invalid AI response")
	}
}
