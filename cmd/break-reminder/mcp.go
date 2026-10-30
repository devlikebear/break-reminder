package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/paths"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: i18n.Text("Connect an MCP client (Claude Code/Desktop) to the running app over stdio"),
		Long: i18n.Text("Relays MCP between stdio and the menu bar app's local endpoint. Register it with: ") +
			"claude mcp add break-reminder -- break-reminder mcp",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCPBridge(cmd.Context(), &mcp.StdioTransport{})
		},
	}
	allowInvalidConfig(cmd)
	return cmd
}

type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	c.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(c)
}

// runMCPBridge pipes every JSON-RPC message between local and the engine's
// HTTP endpoint without interpreting it, so the client sees the engine's
// tools exactly.
func runMCPBridge(ctx context.Context, local mcp.Transport) error {
	notRunning := errors.New(i18n.Text("Break Reminder is not serving MCP. Start the menu bar app, then reconnect."))
	d, err := mcphost.ReadDiscovery(paths.MCPDiscoveryFile())
	if err != nil {
		return notRunning
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	upstream, err := (&mcp.StreamableClientTransport{
		Endpoint:   d.URL,
		HTTPClient: &http.Client{Transport: bearerTransport{token: d.Token}},
	}).Connect(ctx)
	if err != nil {
		return notRunning
	}
	defer upstream.Close()
	client, err := local.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return relay(ctx, client, upstream)
}

func relay(ctx context.Context, client, upstream mcp.Connection) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	pump := func(from, to mcp.Connection, quitIsClean bool) {
		defer wg.Done()
		for {
			msg, err := from.Read(ctx)
			if err == nil {
				err = to.Write(ctx, msg)
			}
			if err != nil {
				if !(quitIsClean && isClosedErr(err)) {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
				_ = from.Close()
				_ = to.Close()
				return
			}
		}
	}
	wg.Add(2)
	go pump(client, upstream, true)
	go pump(upstream, client, true)
	wg.Wait()
	if firstErr != nil && !isClosedErr(firstErr) {
		return firstErr
	}
	return nil
}

func isClosedErr(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) || errors.Is(err, mcp.ErrConnectionClosed) {
		return true
	}
	return strings.Contains(err.Error(), "closed")
}
