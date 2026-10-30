// Package agent is a small tool loop over tars pkg/llm. It deliberately does
// not use tars pkg/agentloop or pkg/session (Experimental there): the loop
// below is all this app needs.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/devlikebear/tars/pkg/llm"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/provider"
)

// MaxToolRounds bounds one question's tool calls.
const MaxToolRounds = 8

// Agent answers questions with the app's MCP tools.
type Agent struct {
	Host *mcphost.Host
}

// Request is one question.
type Request struct {
	Client llm.Client
	// SelfExecuting marks Claude Code, which runs its own loop against an
	// MCP endpoint instead of returning tool calls.
	SelfExecuting bool
	Question      string
	History       []llm.ChatMessage // earlier turns of this conversation
	// AllowActions exposes the tools that change the timer. Off, the model
	// can only propose actions (suggest_action) for the user to click.
	AllowActions bool
	OnDelta      func(string)
}

// Result is the answer and what it took.
type Result struct {
	Text        string
	Suggestions []mcphost.Suggestion
	ToolCalls   []string
	History     []llm.ChatMessage // History + question + answer
}

// SystemPrompt describes the assistant to the model.
func SystemPrompt(allowActions bool) string {
	actions := "You cannot change the timer yourself. When an action would help, call suggest_action so the user gets a button; never claim you did it."
	if allowActions {
		actions = "The user asked you to act: you may use start_pomodoro, start_timer, pause and resume, then say what you did."
	}
	return "You are the assistant inside Break Reminder, a macOS menu bar app that keeps a healthy work/break rhythm. " +
		"Answer with the app's tools (timer status, today's calendar, focus window, app usage, work history); never guess numbers. " +
		"Be brief: at most 3 sentences, concrete times and minutes. " + actions + " " +
		"Reply in " + i18n.ResponseLanguage() + "."
}

type suggestions struct {
	mu   sync.Mutex
	list []mcphost.Suggestion
}

func (s *suggestions) add(x mcphost.Suggestion) {
	s.mu.Lock()
	s.list = append(s.list, x)
	s.mu.Unlock()
}
func (s *suggestions) all() []mcphost.Suggestion {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]mcphost.Suggestion(nil), s.list...)
}

// Run answers req.Question.
func (a *Agent) Run(ctx context.Context, req Request) (Result, error) {
	if req.Client == nil {
		return Result{}, errors.New(i18n.Text("No AI provider is configured."))
	}
	sugg := &suggestions{}
	opts := mcphost.ServerOptions{Actions: req.AllowActions, Suggest: sugg.add}
	msgs := append([]llm.ChatMessage{{Role: "system", Content: SystemPrompt(req.AllowActions)}}, req.History...)
	msgs = append(msgs, llm.ChatMessage{Role: "user", Content: req.Question})

	var res Result
	var err error
	if req.SelfExecuting {
		res, err = a.runSelfExecuting(ctx, req, msgs, opts)
	} else {
		res, err = a.runLoop(ctx, req, msgs, opts)
	}
	if err != nil {
		return Result{}, err
	}
	res.Suggestions = sugg.all()
	res.History = append(append(append([]llm.ChatMessage(nil), req.History...),
		llm.ChatMessage{Role: "user", Content: req.Question}),
		llm.ChatMessage{Role: "assistant", Content: res.Text})
	return res, nil
}

// runLoop: the model returns tool calls, the loop runs them in process.
func (a *Agent) runLoop(ctx context.Context, req Request, msgs []llm.ChatMessage, opts mcphost.ServerOptions) (Result, error) {
	session, closeSession, err := a.Host.ConnectInProcess(ctx, opts)
	if err != nil {
		return Result{}, err
	}
	defer closeSession()
	schemas, err := toolSchemas(ctx, session)
	if err != nil {
		return Result{}, err
	}

	var res Result
	lastCall, repeats := "", 0
	for round := 0; ; round++ {
		tools := schemas
		if round >= MaxToolRounds || repeats >= 2 {
			// Out of budget or going in circles: ask for an answer with what it has.
			tools = nil
			msgs = append(msgs, llm.ChatMessage{Role: "user", Content: "Stop calling tools and answer now with what you have."})
		}
		resp, err := req.Client.Chat(ctx, msgs, llm.ChatOptions{Tools: tools, OnDelta: req.OnDelta})
		if err != nil {
			return Result{}, err
		}
		if len(resp.Message.ToolCalls) == 0 || tools == nil {
			res.Text = strings.TrimSpace(resp.Message.Content)
			return res, nil
		}
		msgs = append(msgs, resp.Message)
		for _, tc := range resp.Message.ToolCalls {
			key := tc.Name + " " + strings.TrimSpace(tc.Arguments)
			if key == lastCall {
				repeats++
			} else {
				lastCall, repeats = key, 0
			}
			res.ToolCalls = append(res.ToolCalls, tc.Name)
			msgs = append(msgs, llm.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: callTool(ctx, session, tc)})
		}
	}
}

// runSelfExecuting: Claude Code calls the tools itself over a loopback MCP
// endpoint that lives only for this question.
func (a *Agent) runSelfExecuting(ctx context.Context, req Request, msgs []llm.ChatMessage, opts mcphost.ServerOptions) (Result, error) {
	var mu sync.Mutex
	var calls []string
	bridge, err := mcphost.StartHTTP(a.Host, opts, "")
	if err != nil {
		return Result{}, err
	}
	defer bridge.Close()
	streamed := false
	resp, err := req.Client.Chat(ctx, msgs, llm.ChatOptions{
		OnDelta: func(s string) {
			streamed = true
			if req.OnDelta != nil {
				req.OnDelta(s)
			}
		},
		OnProviderTool: func(ev llm.ProviderToolEvent) {
			if ev.Finished {
				return
			}
			mu.Lock()
			calls = append(calls, strings.TrimPrefix(ev.Call.Name, "mcp__"+provider.MCPServerName+"__"))
			mu.Unlock()
		},
		ClaudeCodeMCPServers: []llm.ClaudeCodeMCPServer{{
			Name: provider.MCPServerName, Transport: "http", URL: bridge.URL,
			Headers: map[string]string{"Authorization": "Bearer " + bridge.Token},
		}},
		ClaudeCodeHarness: &llm.ClaudeCodeHarnessOptions{MaxTurns: MaxToolRounds + 2},
	})
	if err != nil {
		return Result{}, err
	}
	text := strings.TrimSpace(resp.Message.Content)
	if !streamed && text != "" && req.OnDelta != nil {
		req.OnDelta(text)
	}
	mu.Lock()
	defer mu.Unlock()
	return Result{Text: text, ToolCalls: calls}, nil
}

// Generate is a single call without tools, for briefings and retrospectives
// whose data is already in the prompt.
func Generate(ctx context.Context, client llm.Client, system, prompt string) (string, error) {
	if client == nil {
		return "", errors.New(i18n.Text("No AI provider is configured."))
	}
	resp, err := client.Chat(ctx, []llm.ChatMessage{{Role: "system", Content: system}, {Role: "user", Content: prompt}}, llm.ChatOptions{})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Message.Content), nil
}

func toolSchemas(ctx context.Context, s *mcp.ClientSession) ([]llm.ToolSchema, error) {
	list, err := s.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make([]llm.ToolSchema, 0, len(list.Tools))
	for _, t := range list.Tools {
		params, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, err
		}
		out = append(out, llm.ToolSchema{Type: "function", Function: llm.ToolFunctionSchema{Name: t.Name, Description: t.Description, Parameters: params}})
	}
	return out, nil
}

func callTool(ctx context.Context, s *mcp.ClientSession, tc llm.ToolCall) string {
	var args map[string]any
	if strings.TrimSpace(tc.Arguments) != "" {
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			return fmt.Sprintf("error: arguments are not valid JSON: %v", err)
		}
	}
	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: tc.Name, Arguments: args})
	if err != nil {
		return "error: " + err.Error()
	}
	var b strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	if res.IsError {
		return "error: " + b.String()
	}
	return b.String()
}
