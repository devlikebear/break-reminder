package agent

import (
	"context"
	"errors"
	"slices"

	"github.com/devlikebear/tars/pkg/llm"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/provider"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

var (
	// ErrAIDisabled: ai_enabled is off.
	ErrAIDisabled = errors.New(i18n.Text("AI features are turned off."))
	// ErrNoConsent: the user has not agreed to send their data to this provider.
	ErrNoConsent = errors.New(i18n.Text("Allow this provider to receive your calendar and app usage first."))
)

// Service is the only way the rest of the app reaches a model. It resolves
// the provider from settings on every call (so a changed key or model takes
// effect at once) and keeps llm types inside this package.
type Service struct {
	Host       *mcphost.Host
	LoadConfig func() (config.Config, error)
	Secrets    secrets.Store
	// NewClient builds the client; defaults to provider.New.
	NewClient func(provider.Resolved) (llm.Client, error)
}

// Turn is one message of a conversation.
type Turn struct {
	Role    string `json:"role"` // user | assistant
	Content string `json:"content"`
}

// Answer is the reply to Ask.
type Answer struct {
	Text        string               `json:"text"`
	Suggestions []mcphost.Suggestion `json:"suggestions,omitempty"`
	ToolCalls   []string             `json:"tool_calls,omitempty"`
	History     []Turn               `json:"history"`
}

// ProviderStatus describes the selected provider for the settings UI.
type ProviderStatus struct {
	Enabled       bool     `json:"enabled"`
	Kind          string   `json:"kind"`
	Model         string   `json:"model"`
	Ready         bool     `json:"ready"`
	Consented     bool     `json:"consented"`
	SelfExecuting bool     `json:"self_executing"`
	ShareTitles   bool     `json:"share_event_titles"`
	Models        []string `json:"models,omitempty"`
	Kinds         []string `json:"kinds"`
	Error         string   `json:"error,omitempty"`
}

func (s *Service) newClient(r provider.Resolved) (llm.Client, error) {
	if s.NewClient != nil {
		return s.NewClient(r)
	}
	return provider.New(r)
}

func (s *Service) resolve(needConsent bool) (llm.Client, provider.Resolved, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, provider.Resolved{}, err
	}
	if !cfg.AIEnabled {
		return nil, provider.Resolved{}, ErrAIDisabled
	}
	r, err := provider.Resolve(cfg, s.Secrets)
	if err != nil {
		return nil, r, err
	}
	if needConsent && !r.Consented {
		return nil, r, ErrNoConsent
	}
	c, err := s.newClient(r)
	return c, r, err
}

// Status reports the provider without calling it.
func (s *Service) Status() ProviderStatus {
	cfg, err := s.LoadConfig()
	st := ProviderStatus{Kinds: availableKinds()}
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Enabled = cfg.AIEnabled
	st.ShareTitles = cfg.ShareEventTitles
	st.Model = cfg.AIModel
	kind, err := provider.Kind(cfg)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Kind = kind
	st.SelfExecuting = provider.IsSelfExecuting(kind)
	st.Consented = slices.Contains(cfg.AIConsentedProviders, kind)
	st.Models = provider.ModelChoices(kind)
	r, err := provider.Resolve(cfg, s.Secrets)
	switch {
	case err != nil:
		st.Error = err.Error()
	case kind == config.AIProviderClaudeCode && !provider.CLIAllowed:
		st.Error = provider.ErrCLINotAllowed.Error()
	case kind == config.AIProviderClaudeCode && r.CLIPath == "":
		st.Error = provider.ErrCLINotFound.Error()
	default:
		st.Ready = true
	}
	return st
}

func availableKinds() []string {
	var out []string
	for _, k := range config.AIProviderKinds {
		if k == config.AIProviderClaudeCode && !provider.CLIAllowed {
			continue
		}
		out = append(out, k)
	}
	return out
}

// Ask answers a question with the app's tools. Calendar and app usage reach
// the provider through the tools, so consent is required.
func (s *Service) Ask(ctx context.Context, question string, history []Turn, allowActions bool, onDelta func(string)) (Answer, error) {
	client, r, err := s.resolve(true)
	if err != nil {
		return Answer{}, err
	}
	msgs := make([]llm.ChatMessage, 0, len(history))
	for _, t := range history {
		msgs = append(msgs, llm.ChatMessage{Role: t.Role, Content: t.Content})
	}
	res, err := (&Agent{Host: s.Host}).Run(ctx, Request{
		Client: client, SelfExecuting: provider.IsSelfExecuting(r.Kind),
		Question: question, History: msgs, AllowActions: allowActions, OnDelta: onDelta,
	})
	if err != nil {
		return Answer{}, err
	}
	ans := Answer{Text: res.Text, Suggestions: res.Suggestions, ToolCalls: res.ToolCalls}
	for _, m := range res.History {
		ans.History = append(ans.History, Turn{Role: m.Role, Content: m.Content})
	}
	return ans, nil
}

// Write turns prompt (which carries the user's numbers) into text: one call,
// no tools.
func (s *Service) Write(ctx context.Context, system, prompt string) (string, error) {
	client, _, err := s.resolve(true)
	if err != nil {
		return "", err
	}
	return Generate(ctx, client, system, prompt)
}

// Query is Write without a system prompt, for the `ai` and `insights`
// commands (insights.AIClient).
func (s *Service) Query(ctx context.Context, prompt string) (string, error) {
	return s.Write(ctx, "Reply in "+i18n.ResponseLanguage()+".", prompt)
}

// Test makes a tiny call that carries no user data, so it needs no consent.
func (s *Service) Test(ctx context.Context) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	r, err := provider.Resolve(cfg, s.Secrets)
	if err != nil {
		return err
	}
	c, err := s.newClient(r)
	if err != nil {
		return err
	}
	return provider.TestConnection(ctx, c)
}
