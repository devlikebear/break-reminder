//go:build mas

package provider

import (
	"errors"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestMASBuildRejectsClaudeCode(t *testing.T) {
	_, err := New(Resolved{Kind: config.AIProviderClaudeCode, CLIPath: "/bin/true"})
	if !errors.Is(err, ErrCLINotAllowed) {
		t.Fatalf("err = %v", err)
	}
}
