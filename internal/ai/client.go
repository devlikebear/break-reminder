package ai

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client wraps an AI CLI tool (claude or codex).
type Client struct {
	CLIName string // "claude" or "codex"
	Timeout time.Duration
}

// NewClient creates a new AI client.
func NewClient(cliName string) *Client {
	return &Client{
		CLIName: cliName,
		Timeout: 120 * time.Second,
	}
}

// Available checks if the CLI tool is in PATH.
func (c *Client) Available() bool {
	_, err := exec.LookPath(c.CLIName)
	return err == nil
}

// Query sends a prompt to the AI CLI and returns the response.
func (c *Client) Query(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch c.CLIName {
	case "claude":
		cmd = exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "text", "--max-turns", "1")
	case "codex":
		cmd = exec.CommandContext(ctx, "codex", "-q", prompt)
	default:
		return "", fmt.Errorf("unsupported AI CLI: %s", c.CLIName)
	}

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		var stderr []byte
		if errors.As(err, &exitErr) {
			stderr = exitErr.Stderr
		}
		return "", formatQueryError(ctx, c.CLIName, err, stderr, out)
	}

	return strings.TrimSpace(string(out)), nil
}

// formatQueryError keeps the useful diagnostic emitted by the AI CLI. The
// dashboard displays this error verbatim, so a failed refresh is actionable
// instead of looking like an empty result.
func formatQueryError(ctx context.Context, cliName string, commandErr error, stderr, stdout []byte) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("AI CLI %q timed out", cliName)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("AI CLI %q was canceled", cliName)
	}

	detail := strings.TrimSpace(string(stderr))
	if detail == "" {
		detail = strings.TrimSpace(string(stdout))
	}
	if detail != "" {
		return fmt.Errorf("AI CLI %q error: %s: %w", cliName, detail, commandErr)
	}
	return fmt.Errorf("AI CLI %q error: %w", cliName, commandErr)
}
