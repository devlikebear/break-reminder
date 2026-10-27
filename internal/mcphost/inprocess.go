package mcphost

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConnectInProcess returns a client session talking to a fresh server over
// in-memory transports, for the built-in agent. close releases both ends.
func (h *Host) ConnectInProcess(ctx context.Context, opts ServerOptions) (session *mcp.ClientSession, close func(), err error) {
	ct, st := mcp.NewInMemoryTransports()
	ss, err := h.Server(opts).Connect(ctx, st, nil)
	if err != nil {
		return nil, nil, err
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "break-reminder-agent", Version: ServerVersion}, nil).Connect(ctx, ct, nil)
	if err != nil {
		_ = ss.Close()
		return nil, nil, err
	}
	return cs, func() { _ = cs.Close(); _ = ss.Close() }, nil
}
