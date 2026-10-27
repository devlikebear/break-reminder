package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/provider"
)

// Runs the real `claude` CLI (the user's subscription) against fixture data.
// BREAK_REMINDER_LIVE_CLAUDE=1 go test ./internal/agent -run Live -v
func TestLiveClaudeCodeUsesOurTools(t *testing.T) {
	if os.Getenv("BREAK_REMINDER_LIVE_CLAUDE") != "1" {
		t.Skip("set BREAK_REMINDER_LIVE_CLAUDE=1 to call Claude Code")
	}
	r := provider.Resolved{Kind: config.AIProviderClaudeCode, Model: "haiku", CLIPath: provider.FindClaudeCLI()}
	client, err := provider.New(r)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	b := &backend{}
	res, err := (&Agent{Host: &mcphost.Host{Backend: b}}).Run(ctx, Request{
		Client: client, SelfExecuting: true,
		Question: "How many minutes have I worked in the current block, and how many pomodoros fit before my next event?",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("answer: %s\ntools: %v\nsuggestions: %+v", res.Text, res.ToolCalls, res.Suggestions)
	if !strings.Contains(res.Text, "20") || len(res.ToolCalls) == 0 {
		t.Fatalf("expected an answer built from the tools (20 min worked), got %q via %v", res.Text, res.ToolCalls)
	}
	if len(b.calls) != 0 {
		t.Fatalf("actions ran: %v", b.calls)
	}
}
