package main

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

type bridgeBackend struct{}

func (bridgeBackend) Timer() (mcphost.TimerView, error) {
	return mcphost.TimerView{Phase: "working", WorkMin: 12}, nil
}
func (bridgeBackend) Calendar() (mcphost.CalendarView, error)      { return mcphost.CalendarView{}, nil }
func (bridgeBackend) AppUsage(int) ([]workcontext.AppUsage, error) { return nil, nil }
func (bridgeBackend) WorkHistory(int) ([]ai.DailySummary, error)   { return nil, nil }
func (bridgeBackend) ShareEventTitles() bool                       { return false }
func (bridgeBackend) StartPomodoro() error                         { return nil }
func (bridgeBackend) StartTimer(int, string) error                 { return nil }
func (bridgeBackend) Pause(string, int) error                      { return nil }
func (bridgeBackend) Resume() error                                { return nil }

func TestMCPBridgeRelaysToEngine(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BREAK_REMINDER_HOME", root)
	srv, err := mcphost.StartHTTP(&mcphost.Host{Backend: bridgeBackend{}}, mcphost.ServerOptions{}, filepath.Join(root, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// client <-> bridge over a pair of pipes, as Claude Code would over stdio.
	toBridgeR, toBridgeW := io.Pipe()
	fromBridgeR, fromBridgeW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runMCPBridge(ctx, &mcp.IOTransport{Reader: toBridgeR, Writer: fromBridgeW}) }()

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "claude-code", Version: "0"}, nil).
		Connect(ctx, &mcp.IOTransport{Reader: fromBridgeR, Writer: toBridgeW}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_status"})
	if err != nil || res.IsError {
		t.Fatalf("call = %+v, %v", res, err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; !strings.Contains(text, `"work_min":12`) {
		t.Fatalf("text = %s", text)
	}
	cs.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not stop after the client left")
	}
}

func TestMCPBridgeExplainsWhenEngineIsDown(t *testing.T) {
	t.Setenv("BREAK_REMINDER_HOME", t.TempDir())
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	err := runMCPBridge(context.Background(), &mcp.IOTransport{Reader: io.NopCloser(strings.NewReader("")), Writer: nopWriteCloser{io.Discard}})
	if err == nil || !strings.Contains(err.Error(), "menu bar app") {
		t.Fatalf("err = %v", err)
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
