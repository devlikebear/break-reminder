package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/engine"
	"github.com/devlikebear/break-reminder/internal/enginelock"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/mcphost"
	"github.com/devlikebear/break-reminder/internal/paths"
	"github.com/devlikebear/break-reminder/internal/reminder"
)

func newEngineCmd() *cobra.Command {
	var stdio bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:    "engine",
		Short:  i18n.Text("Run the timer engine for the menu bar app (JSON-RPC over stdio)"),
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !stdio {
				return errors.New(i18n.Text("--stdio is required"))
			}
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return runEngine(ctx, os.Stdin, os.Stdout, interval)
		},
	}
	cmd.Flags().BoolVar(&stdio, "stdio", false, i18n.Text("Serve JSON-RPC on stdin/stdout"))
	cmd.Flags().DurationVar(&interval, "interval", 0, i18n.Text("Tick interval (default: check_interval_sec from config)"))
	allowInvalidConfig(cmd)
	return cmd
}

// runEngine serves until in closes, which is how the engine follows its parent
// app: when the app dies, stdin reaches EOF and the engine exits.
func runEngine(ctx context.Context, in io.Reader, out io.Writer, interval time.Duration) error {
	lock, err := enginelock.Acquire(paths.EngineLockFile())
	if err != nil {
		return err
	}
	defer lock.Release()

	if interval <= 0 {
		interval = 60 * time.Second
		if c, err := loadAppConfig(); err == nil && c.CheckIntervalSec > 0 {
			interval = time.Duration(c.CheckIntervalSec) * time.Second
		}
	}

	server := engine.NewServer(out)
	host := &mcphost.Host{}
	eng := engine.New(engine.Options{
		LoadConfig: loadAppConfig,
		Emit:       func(e reminder.Event) { _ = server.Notify(engine.NotifyEvent, e) },
		Status:     func(s engine.Status) { _ = server.Notify(engine.NotifyStatus, s) },
		Notify:     func(method string, params any) { _ = server.Notify(method, params) },
		AI:         newAIService(host),
	})
	host.Backend = eng.Backend()
	eng.Register(server, version)

	// External MCP clients (Claude Code/Desktop via `break-reminder mcp`).
	if mcpServer, err := mcphost.StartHTTP(host, mcphost.ServerOptions{Actions: true}, paths.MCPDiscoveryFile()); err != nil {
		logging.Log(logging.DefaultLogPath(), i18n.Text("MCP endpoint unavailable")+": "+err.Error())
	} else {
		defer mcpServer.Close()
	}

	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	logging.Log(logging.DefaultLogPath(), fmt.Sprintf(i18n.Text("Engine started (pid %d)"), os.Getpid()))
	workers.Add(2)
	go func() { defer workers.Done(); eng.Run(ctx, interval) }()
	go func() { defer workers.Done(); eng.RunTimeTools(ctx) }()

	err = server.Serve(ctx, in)
	eng.WaitBackground()
	logging.Log(logging.DefaultLogPath(), i18n.Text("Engine stopped"))
	return err
}
