package agent

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/devlikebear/tars/pkg/llm"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

func TestMain(m *testing.M) {
	os.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	os.Exit(m.Run())
}

type backend struct{ calls []string }

func (b *backend) Timer() (mcphost.TimerView, error) {
	return mcphost.TimerView{Phase: "working", WorkMin: 20, WorkTargetMin: 50}, nil
}
func (b *backend) Calendar() (mcphost.CalendarView, error) {
	return mcphost.CalendarView{FocusMinutes: 55, FocusPomodoros: 2}, nil
}
func (b *backend) AppUsage(int) ([]workcontext.AppUsage, error) { return nil, nil }
func (b *backend) WorkHistory(int) ([]ai.DailySummary, error)  { return nil, nil }
func (b *backend) ShareEventTitles() bool                      { return false }
func (b *backend) StartPomodoro() error                        { b.calls = append(b.calls, "pomodoro"); return nil }
func (b *backend) StartTimer(int, string) error                { b.calls = append(b.calls, "timer"); return nil }
func (b *backend) Pause(string, int) error                     { b.calls = append(b.calls, "pause"); return nil }
func (b *backend) Resume() error                               { b.calls = append(b.calls, "resume"); return nil }

// scripted answers each Chat with the next scripted response and records
// what it was sent.
type scripted struct {
	mu      sync.Mutex
	replies []llm.ChatResponse
	seen    [][]llm.ChatMessage
	tools   [][]string
}

func (s *scripted) Ask(ctx context.Context, p string) (string, error) { return "", errors.New("unused") }
func (s *scripted) Chat(_ context.Context, msgs []llm.ChatMessage, o llm.ChatOptions) (llm.ChatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, append([]llm.ChatMessage(nil), msgs...))
	var names []string
	for _, t := range o.Tools {
		names = append(names, t.Function.Name)
	}
	s.tools = append(s.tools, names)
	if len(s.replies) == 0 {
		return llm.ChatResponse{Message: llm.ChatMessage{Role: "assistant", Content: "done"}}, nil
	}
	r := s.replies[0]
	s.replies = s.replies[1:]
	if o.OnDelta != nil && r.Message.Content != "" {
		o.OnDelta(r.Message.Content)
	}
	return r, nil
}

func call(id, name, args string) llm.ChatResponse {
	return llm.ChatResponse{Message: llm.ChatMessage{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: id, Name: name, Arguments: args}}}}
}
func say(text string) llm.ChatResponse {
	return llm.ChatResponse{Message: llm.ChatMessage{Role: "assistant", Content: text}}
}

func TestLoopRunsToolsThenAnswers(t *testing.T) {
	b := &backend{}
	client := &scripted{replies: []llm.ChatResponse{
		call("1", "get_status", `{}`),
		call("2", "get_focus_window", `{}`),
		say("You have 55 minutes: two pomodoros fit."),
	}}
	var streamed strings.Builder
	res, err := (&Agent{Host: &mcphost.Host{Backend: b}}).Run(context.Background(), Request{
		Client: client, Question: "What should I do now?", OnDelta: func(s string) { streamed.WriteString(s) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "You have 55 minutes: two pomodoros fit." || streamed.String() != res.Text {
		t.Fatalf("result = %+v streamed=%q", res, streamed.String())
	}
	if strings.Join(res.ToolCalls, ",") != "get_status,get_focus_window" {
		t.Fatalf("tool calls = %v", res.ToolCalls)
	}
	// The second request carries the first tool's result.
	second := client.seen[1]
	last := second[len(second)-1]
	if last.Role != "tool" || last.ToolCallID != "1" || !strings.Contains(last.Content, `"phase":"working"`) {
		t.Fatalf("tool result message = %+v", last)
	}
	// Read-only by default: no action tools offered, suggestions are.
	offered := strings.Join(client.tools[0], ",")
	if strings.Contains(offered, "start_pomodoro") || !strings.Contains(offered, mcphost.SuggestTool) {
		t.Fatalf("offered tools = %s", offered)
	}
	if len(b.calls) != 0 {
		t.Fatalf("actions ran: %v", b.calls)
	}
}

func TestSuggestionsComeBackAsButtons(t *testing.T) {
	b := &backend{}
	client := &scripted{replies: []llm.ChatResponse{
		call("1", mcphost.SuggestTool, `{"action":"start_pomodoro","reason":"two blocks fit"}`),
		say("Want to start?"),
	}}
	res, err := (&Agent{Host: &mcphost.Host{Backend: b}}).Run(context.Background(), Request{Client: client, Question: "now?"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suggestions) != 1 || res.Suggestions[0].Action != "start_pomodoro" || len(b.calls) != 0 {
		t.Fatalf("suggestions = %+v calls=%v", res.Suggestions, b.calls)
	}
}

func TestActionsOnlyWhenAllowed(t *testing.T) {
	b := &backend{}
	client := &scripted{replies: []llm.ChatResponse{call("1", "start_pomodoro", `{}`), say("Started.")}}
	if _, err := (&Agent{Host: &mcphost.Host{Backend: b}}).Run(context.Background(), Request{Client: client, Question: "start a pomodoro", AllowActions: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(b.calls, ",") != "pomodoro" {
		t.Fatalf("calls = %v", b.calls)
	}
	// Not allowed: the call fails as an unknown tool and nothing runs.
	b2 := &backend{}
	client2 := &scripted{replies: []llm.ChatResponse{call("1", "start_pomodoro", `{}`), say("I can't.")}}
	if _, err := (&Agent{Host: &mcphost.Host{Backend: b2}}).Run(context.Background(), Request{Client: client2, Question: "start"}); err != nil {
		t.Fatal(err)
	}
	if len(b2.calls) != 0 {
		t.Fatalf("action ran without permission: %v", b2.calls)
	}
}

func TestLoopStopsOnRepeatsAndCap(t *testing.T) {
	var replies []llm.ChatResponse
	for i := 0; i < 20; i++ {
		replies = append(replies, call("x", "get_status", `{}`))
	}
	client := &scripted{replies: replies}
	res, err := (&Agent{Host: &mcphost.Host{Backend: &backend{}}}).Run(context.Background(), Request{Client: client, Question: "loop"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ToolCalls) > MaxToolRounds || len(client.seen) > MaxToolRounds+1 {
		t.Fatalf("ran %d tool calls over %d requests", len(res.ToolCalls), len(client.seen))
	}
	if len(res.ToolCalls) >= MaxToolRounds {
		t.Fatalf("identical repeats should stop early, got %d calls", len(res.ToolCalls))
	}
	// The final request offers no tools, asking for a wrap-up.
	if last := client.tools[len(client.tools)-1]; len(last) != 0 {
		t.Fatalf("wrap-up offered tools %v", last)
	}
}

func TestHistoryIsKept(t *testing.T) {
	client := &scripted{replies: []llm.ChatResponse{say("second answer")}}
	history := []llm.ChatMessage{{Role: "user", Content: "first"}, {Role: "assistant", Content: "first answer"}}
	res, err := (&Agent{Host: &mcphost.Host{Backend: &backend{}}}).Run(context.Background(), Request{Client: client, Question: "second", History: history})
	if err != nil {
		t.Fatal(err)
	}
	sent := client.seen[0]
	if sent[0].Role != "system" || sent[1].Content != "first" || sent[len(sent)-1].Content != "second" {
		t.Fatalf("sent = %+v", sent)
	}
	if len(res.History) != 4 || res.History[3].Content != "second answer" {
		t.Fatalf("history = %+v", res.History)
	}
}

func TestGenerateIsOneCallWithoutTools(t *testing.T) {
	client := &scripted{replies: []llm.ChatResponse{say("  Good morning.  ")}}
	text, err := Generate(context.Background(), client, "system", "prompt")
	if err != nil || text != "Good morning." || len(client.seen) != 1 || len(client.tools[0]) != 0 {
		t.Fatalf("text=%q err=%v seen=%d", text, err, len(client.seen))
	}
}
