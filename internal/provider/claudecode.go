package provider

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/devlikebear/tars/pkg/llm"
)

// Claude Code is a whole agent with file, shell and web tools plus the
// user's own hooks, plugins and MCP servers. Here it only needs this app's
// tools, so every client leaves this package wrapped in lockedClaudeCode: no
// built-in tools, none of the user's settings, no MCP server but ours, and
// every permission prompt refused unless it is for one of our tools. The
// wrapper takes nothing from the caller that could widen that.

const claudeCodeCLIPathEnv = "CLAUDE_CODE_CLI_PATH"

// MCPServerName is the server name the agent registers with Claude Code.
const MCPServerName = "break_reminder"

const claudeCodeToolPrefix = "mcp__" + MCPServerName + "__"

type lockedClaudeCode struct{ inner llm.Client }

func (c lockedClaudeCode) Ask(ctx context.Context, prompt string) (string, error) {
	resp, err := c.Chat(ctx, []llm.ChatMessage{{Role: "user", Content: prompt}}, llm.ChatOptions{})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Message.Content), nil
}

func (c lockedClaudeCode) Chat(ctx context.Context, messages []llm.ChatMessage, opts llm.ChatOptions) (llm.ChatResponse, error) {
	maxTurns := 0
	if opts.ClaudeCodeHarness != nil {
		maxTurns = opts.ClaudeCodeHarness.MaxTurns
	}
	locked := llm.ChatOptions{
		OnDelta:                     opts.OnDelta,
		OnProviderTool:              opts.OnProviderTool,
		ClaudeCodeMCPServers:        ourServerOnly(opts.ClaudeCodeMCPServers),
		ClaudeCodePermissionMode:    "default",
		ClaudeCodePermissionHandler: onlyOurTools,
		ClaudeCodeHarness: &llm.ClaudeCodeHarnessOptions{
			Restricted:         true,
			StrictMCP:          true,
			DisableChrome:      true,
			IsolateEnvironment: true,
			Tools:              []string{}, // `--tools ""`: no built-in tool at all
			MaxTurns:           maxTurns,
		},
	}
	return c.inner.Chat(ctx, instructionsAsPrompt(messages), locked)
}

func ourServerOnly(servers []llm.ClaudeCodeMCPServer) []llm.ClaudeCodeMCPServer {
	var out []llm.ClaudeCodeMCPServer
	for _, s := range servers {
		if s.Name == MCPServerName {
			out = append(out, s)
		}
	}
	return out
}

func onlyOurTools(_ context.Context, req llm.ClaudeCodePermissionRequest) (llm.ClaudeCodePermissionDecision, error) {
	if strings.HasPrefix(req.ToolName, claudeCodeToolPrefix) {
		return llm.ClaudeCodePermissionDecision{Allow: true}, nil
	}
	return llm.ClaudeCodePermissionDecision{Message: "Only Break Reminder's own tools (" + claudeCodeToolPrefix + "*) are available here."}, nil
}

// instructionsAsPrompt moves the system message into the conversation, which
// tars sends over stdin rather than the command line.
func instructionsAsPrompt(messages []llm.ChatMessage) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(messages))
	for _, m := range messages {
		if strings.EqualFold(strings.TrimSpace(m.Role), "system") {
			if text := strings.TrimSpace(m.Content); text != "" {
				out = append(out, llm.ChatMessage{Role: "user", Content: "[Instructions from Break Reminder, the app you are working inside. Follow them as your system instructions.]\n\n" + text})
			}
			continue
		}
		out = append(out, m)
	}
	return out
}

// pointTarsAtCLI makes tars run the detected binary, with its directory on
// PATH so an npm install finds `node` next to it.
func pointTarsAtCLI(path string) {
	_ = os.Setenv(claudeCodeCLIPathEnv, path)
	dir := filepath.Dir(path)
	current := os.Getenv("PATH")
	if slices.Contains(filepath.SplitList(current), dir) {
		return
	}
	if current == "" {
		_ = os.Setenv("PATH", dir)
		return
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+current)
}

// loginShellClaude asks the user's login shell, which sees their full PATH.
func loginShellClaude() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-lc", "command -v claude").Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if isExecutable(p) {
		return p
	}
	return ""
}
