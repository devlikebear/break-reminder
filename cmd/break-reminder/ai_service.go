package main

import (
	"errors"

	"github.com/devlikebear/break-reminder/internal/agent"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

// newAIService builds the agent service over the current settings. host may
// be nil for callers that only Query/Write (no tools).
func newAIService(host *mcphost.Host) *agent.Service {
	return &agent.Service{Host: host, LoadConfig: loadAppConfig, Secrets: secrets.Default}
}

// readyAIService returns the service when the selected provider is usable.
func readyAIService() (*agent.Service, error) {
	svc := newAIService(nil)
	if st := svc.Status(); !st.Ready {
		if !st.Enabled {
			return nil, agent.ErrAIDisabled
		}
		return nil, errors.New(st.Error)
	}
	return svc, nil
}
