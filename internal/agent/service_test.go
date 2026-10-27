package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/devlikebear/tars/pkg/llm"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/provider"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

func service(cfg config.Config, client *scripted) (*Service, *config.Config) {
	c := cfg
	s := &Service{
		Host:       &mcphost.Host{Backend: &backend{}},
		LoadConfig: func() (config.Config, error) { return c, nil },
		Secrets:    secrets.NewMemory(),
		NewClient:  func(provider.Resolved) (llm.Client, error) { return client, nil },
	}
	return s, &c
}

func consented() config.Config {
	cfg := config.Default()
	cfg.AIEnabled = true
	cfg.AIProvider = config.AIProviderClaudeCode
	cfg.AIConsentedProviders = []string{config.AIProviderClaudeCode}
	return cfg
}

func TestServiceGates(t *testing.T) {
	client := &scripted{replies: []llm.ChatResponse{say("ok")}}
	cfg := consented()
	cfg.AIEnabled = false
	s, _ := service(cfg, client)
	if _, err := s.Write(context.Background(), "sys", "p"); !errors.Is(err, ErrAIDisabled) {
		t.Fatalf("disabled -> %v", err)
	}
	cfg = consented()
	cfg.AIConsentedProviders = nil
	s, _ = service(cfg, client)
	if _, err := s.Write(context.Background(), "sys", "p"); !errors.Is(err, ErrNoConsent) {
		t.Fatalf("no consent -> %v", err)
	}
	if _, err := s.Ask(context.Background(), "q", nil, false, nil); !errors.Is(err, ErrNoConsent) {
		t.Fatalf("ask without consent -> %v", err)
	}
	if len(client.seen) != 0 {
		t.Fatal("a provider was called without consent")
	}
	// The connection test sends no user data, so it needs no consent.
	client.replies = []llm.ChatResponse{say("OK")}
	if err := s.Test(context.Background()); err != nil {
		t.Fatalf("test -> %v", err)
	}
}

func TestServiceAskAndStatus(t *testing.T) {
	client := &scripted{replies: []llm.ChatResponse{call("1", "get_status", `{}`), say("20 minutes in.")}}
	cfg := consented()
	cfg.AIProvider = config.AIProviderAnthropic
	cfg.AIConsentedProviders = []string{config.AIProviderAnthropic}
	s, _ := service(cfg, client)
	_ = s.Secrets.Set("anthropic_api_key", "sk-test")
	ans, err := s.Ask(context.Background(), "how long?", []Turn{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "hello"}}, false, nil)
	if err != nil || ans.Text != "20 minutes in." || len(ans.History) != 4 {
		t.Fatalf("ask = %+v %v", ans, err)
	}
	st := s.Status()
	if !st.Ready || !st.Consented || st.Kind != config.AIProviderAnthropic || st.SelfExecuting {
		t.Fatalf("status = %+v", st)
	}
}

func TestServiceStatusReportsMissingKey(t *testing.T) {
	cfg := consented()
	cfg.AIProvider = config.AIProviderAnthropic
	s, _ := service(cfg, &scripted{})
	st := s.Status()
	if st.Ready || st.Error == "" {
		t.Fatalf("status = %+v", st)
	}
}
