// Package provider turns settings into a tars llm.Client. It and
// internal/agent are the only packages allowed to import tars pkg/llm
// (scripts/check-llm-deps.sh enforces it), so the tool layer and the timer
// stay model-free.
package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/devlikebear/tars/pkg/llm"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

var (
	// ErrCodexUnsupported: the old `ai_cli: codex` path is gone.
	ErrCodexUnsupported = errors.New(i18n.Text("The codex CLI is no longer supported. Choose Claude Code or an API key provider."))
	// ErrNoAPIKey: an API provider is selected without a key.
	ErrNoAPIKey = errors.New(i18n.Text("No API key is stored for this provider."))
	// ErrCLINotAllowed: the App Store build cannot run Claude Code.
	ErrCLINotAllowed = errors.New(i18n.Text("Claude Code is not available in the App Store version. Use an API key provider."))
	// ErrCLINotFound: the `claude` binary could not be found.
	ErrCLINotFound = errors.New(i18n.Text("Claude Code (claude) was not found. Install it or choose an API key provider."))
)

// Resolved is one provider's effective configuration.
type Resolved struct {
	Kind      string
	Model     string
	APIKey    string
	CLIPath   string // claude-code-cli only
	WorkDir   string // claude-code-cli only: an empty directory the CLI runs in
	Consented bool   // the user agreed to send calendar and app usage data
}

// String redacts the key; Resolved ends up in logs.
func (r Resolved) String() string {
	key := "unset"
	if r.APIKey != "" {
		key = "set"
	}
	return fmt.Sprintf("provider.Resolved{Kind:%s Model:%s APIKey:%s CLIPath:%s Consented:%v}", r.Kind, r.Model, key, r.CLIPath, r.Consented)
}

// SecretName is the keychain entry holding a provider's API key.
func SecretName(kind string) string {
	switch kind {
	case config.AIProviderAnthropic:
		return "anthropic_api_key"
	case config.AIProviderOpenAI:
		return "openai_api_key"
	case config.AIProviderGemini:
		return "gemini_api_key"
	}
	return ""
}

// Kind returns the provider kind settings select. An unset ai_provider
// follows the legacy ai_cli: "claude" (the default) means Claude Code.
func Kind(cfg config.Config) (string, error) {
	if cfg.AIProvider != "" {
		return cfg.AIProvider, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.AICLI)) {
	case "", "claude":
		return config.AIProviderClaudeCode, nil
	case "codex":
		return "", ErrCodexUnsupported
	}
	return "", fmt.Errorf(i18n.Text("unsupported AI CLI: %s"), cfg.AICLI)
}

// Resolve gathers what New needs, reading the key from store.
func Resolve(cfg config.Config, store secrets.Store) (Resolved, error) {
	kind, err := Kind(cfg)
	if err != nil {
		return Resolved{}, err
	}
	r := Resolved{Kind: kind, Model: strings.TrimSpace(cfg.AIModel), Consented: slices.Contains(cfg.AIConsentedProviders, kind)}
	if kind == config.AIProviderClaudeCode {
		r.CLIPath = FindClaudeCLI()
		return r, nil
	}
	key, err := store.Get(SecretName(kind))
	if err != nil || strings.TrimSpace(key) == "" {
		return r, ErrNoAPIKey
	}
	r.APIKey = strings.TrimSpace(key)
	return r, nil
}

// IsSelfExecuting reports whether a provider runs its own tool loop, so the
// agent hands it tools over MCP instead of running ToolCalls itself.
func IsSelfExecuting(kind string) bool { return kind == config.AIProviderClaudeCode }

// ModelChoices offers a few model names per provider; any id still works.
func ModelChoices(kind string) []string {
	switch kind {
	case config.AIProviderClaudeCode:
		return []string{"haiku", "sonnet", "opus"}
	case config.AIProviderAnthropic:
		return []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"}
	}
	return nil
}

var (
	factoryMu sync.Mutex
	factory   = llm.NewProvider
)

// SetClientFactory replaces llm.NewProvider (tests); it returns a restore func.
func SetClientFactory(f func(llm.ProviderOptions) (llm.Client, error)) func() {
	factoryMu.Lock()
	prev := factory
	factory = f
	factoryMu.Unlock()
	return func() { factoryMu.Lock(); factory = prev; factoryMu.Unlock() }
}

// New builds the client. Claude Code comes back locked down (see claudecode.go).
func New(r Resolved) (llm.Client, error) {
	opts := llm.ProviderOptions{Provider: r.Kind, Model: r.Model}
	if r.Kind == config.AIProviderClaudeCode {
		if !CLIAllowed {
			return nil, ErrCLINotAllowed
		}
		if r.CLIPath == "" {
			return nil, ErrCLINotFound
		}
		pointTarsAtCLI(r.CLIPath)
		if r.WorkDir == "" {
			dir, err := os.MkdirTemp("", "break-reminder-claude-")
			if err != nil {
				return nil, err
			}
			r.WorkDir = dir
		}
		opts.WorkDir = r.WorkDir
	} else {
		opts.APIKey = r.APIKey
	}
	factoryMu.Lock()
	f := factory
	factoryMu.Unlock()
	client, err := f(opts)
	if err != nil {
		return nil, err
	}
	if r.Kind == config.AIProviderClaudeCode {
		return lockedClaudeCode{inner: client}, nil
	}
	return client, nil
}

// TestConnection makes one tiny call to prove the provider answers.
func TestConnection(ctx context.Context, c llm.Client) error {
	reply, err := c.Ask(ctx, "Reply with exactly: OK")
	if err != nil {
		return err
	}
	if strings.TrimSpace(reply) == "" {
		return errors.New(i18n.Text("The provider returned an empty reply."))
	}
	return nil
}

// FindClaudeCLI locates the `claude` binary. An app opened from the Finder
// has a PATH without Homebrew or npm, so known install places and the login
// shell are checked too.
func FindClaudeCLI() string {
	if p := os.Getenv(claudeCodeCLIPathEnv); p != "" && isExecutable(p) {
		return p
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".claude", "local", "claude"),
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
	}
	return findClaudeCLIIn(candidates, loginShellClaude)
}

func findClaudeCLIIn(candidates []string, fallback func() string) string {
	for _, c := range candidates {
		if isExecutable(c) {
			return c
		}
	}
	return fallback()
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}
