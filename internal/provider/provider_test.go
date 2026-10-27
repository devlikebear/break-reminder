package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devlikebear/tars/pkg/llm"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

func TestMain(m *testing.M) {
	os.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	os.Exit(m.Run())
}

func TestResolveMapsLegacyAICLI(t *testing.T) {
	store := secrets.NewMemory()
	cfg := config.Default()
	cfg.AICLI = "claude"
	r, err := Resolve(cfg, store)
	if err != nil || r.Kind != config.AIProviderClaudeCode {
		t.Fatalf("claude -> %+v, %v", r, err)
	}
	cfg.AICLI = "codex"
	if _, err := Resolve(cfg, store); !errors.Is(err, ErrCodexUnsupported) {
		t.Fatalf("codex -> %v", err)
	}
	cfg.AIProvider = config.AIProviderAnthropic
	if _, err := Resolve(cfg, store); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("anthropic without key -> %v", err)
	}
	_ = store.Set("anthropic_api_key", "sk-test")
	cfg.AIModel = "claude-haiku-5-5"
	cfg.AIConsentedProviders = []string{config.AIProviderAnthropic}
	r, err = Resolve(cfg, store)
	if err != nil || r.APIKey != "sk-test" || r.Model != "claude-haiku-5-5" || !r.Consented {
		t.Fatalf("anthropic -> %+v, %v", r, err)
	}
	if strings.Contains(r.String(), "sk-test") {
		t.Fatal("String() leaks the key")
	}
}

type fakeClient struct {
	opts  llm.ChatOptions
	msgs  []llm.ChatMessage
	reply string
}

func (f *fakeClient) Ask(ctx context.Context, p string) (string, error) { return f.reply, nil }
func (f *fakeClient) Chat(_ context.Context, m []llm.ChatMessage, o llm.ChatOptions) (llm.ChatResponse, error) {
	f.msgs, f.opts = m, o
	return llm.ChatResponse{Message: llm.ChatMessage{Role: "assistant", Content: f.reply}}, nil
}

func TestNewUsesFactoryWithTarsOptions(t *testing.T) {
	var got llm.ProviderOptions
	restore := SetClientFactory(func(o llm.ProviderOptions) (llm.Client, error) { got = o; return &fakeClient{}, nil })
	defer restore()
	if _, err := New(Resolved{Kind: config.AIProviderAnthropic, APIKey: "k", Model: "m"}); err != nil {
		t.Fatal(err)
	}
	if got.Provider != "anthropic" || got.APIKey != "k" || got.Model != "m" {
		t.Fatalf("opts = %+v", got)
	}
}

func TestClaudeCodeIsLockedDown(t *testing.T) {
	inner := &fakeClient{reply: "hi"}
	restore := SetClientFactory(func(o llm.ProviderOptions) (llm.Client, error) {
		if o.Provider != "claude-code-cli" || o.APIKey != "" || o.WorkDir == "" {
			t.Errorf("claude-code opts = %+v", o)
		}
		return inner, nil
	})
	defer restore()
	if !CLIAllowed {
		t.Skip("mas build")
	}
	c, err := New(Resolved{Kind: config.AIProviderClaudeCode, CLIPath: "/bin/true", WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ours := llm.ClaudeCodeMCPServer{Name: "break_reminder", Transport: "http", URL: "http://127.0.0.1:1/mcp"}
	other := llm.ClaudeCodeMCPServer{Name: "github", Transport: "http", URL: "https://example.com"}
	_, err = c.Chat(context.Background(),
		[]llm.ChatMessage{{Role: "system", Content: "be brief"}, {Role: "user", Content: "hello"}},
		llm.ChatOptions{ClaudeCodeMCPServers: []llm.ClaudeCodeMCPServer{ours, other}, Tools: []llm.ToolSchema{{}}})
	if err != nil {
		t.Fatal(err)
	}
	h := inner.opts.ClaudeCodeHarness
	if h == nil || !h.Restricted || !h.StrictMCP || !h.IsolateEnvironment || !h.DisableChrome || h.Tools == nil || len(h.Tools) != 0 {
		t.Fatalf("harness = %+v", h)
	}
	if len(inner.opts.ClaudeCodeMCPServers) != 1 || inner.opts.ClaudeCodeMCPServers[0].Name != "break_reminder" {
		t.Fatalf("servers = %+v", inner.opts.ClaudeCodeMCPServers)
	}
	if inner.opts.ClaudeCodePermissionHandler == nil || inner.opts.ClaudeCodePermissionMode != "default" {
		t.Fatal("permission handler missing")
	}
	for name, allow := range map[string]bool{"mcp__break_reminder__get_status": true, "Bash": false, "mcp__github__create_issue": false} {
		d, _ := inner.opts.ClaudeCodePermissionHandler(context.Background(), llm.ClaudeCodePermissionRequest{ToolName: name})
		if d.Allow != allow {
			t.Errorf("%s allowed=%v, want %v", name, d.Allow, allow)
		}
	}
	if inner.msgs[0].Role != "user" || !strings.Contains(inner.msgs[0].Content, "be brief") {
		t.Fatalf("system message not moved into the prompt: %+v", inner.msgs)
	}
}

func TestFindClaudeCLIPrefersEnvThenKnownPaths(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CODE_CLI_PATH", bin)
	if got := FindClaudeCLI(); got != bin {
		t.Fatalf("env -> %q", got)
	}
	t.Setenv("CLAUDE_CODE_CLI_PATH", filepath.Join(dir, "missing"))
	got := findClaudeCLIIn([]string{filepath.Join(dir, "nope"), bin}, func() string { return "" })
	if got != bin {
		t.Fatalf("candidates -> %q", got)
	}
	if got := findClaudeCLIIn(nil, func() string { return "" }); got != "" {
		t.Fatalf("nothing -> %q", got)
	}
}

func TestTestConnection(t *testing.T) {
	if err := TestConnection(context.Background(), &fakeClient{reply: "OK"}); err != nil {
		t.Fatal(err)
	}
	if err := TestConnection(context.Background(), &fakeClient{reply: ""}); err == nil {
		t.Fatal("empty reply should fail")
	}
}
