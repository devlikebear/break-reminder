package ai

import (
	"context"
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
		Timeout: 30 * time.Second,
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
		cmd = exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "text")
	case "codex":
		cmd = exec.CommandContext(ctx, "codex", "-q", prompt)
	default:
		return "", fmt.Errorf("unsupported AI CLI: %s", c.CLIName)
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("AI CLI error: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
