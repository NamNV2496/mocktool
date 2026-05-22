// Package mcpserver hosts the MCP-over-SSE adapter for the tools registry.
//
// This binary exposes the same `internal/tools` registry that the Slack bot
// uses, so any MCP-compatible client (Claude Desktop, IDE plugins, custom
// agents) can call mocktool admin operations via the standard MCP protocol.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/caarlos0/env/v6"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"

	"github.com/namnv2496/mocktool/internal/tools"
)

// Config is the mcpserver-specific configuration. It piggybacks on the existing
// AppConfig wiring via fx; values come from env vars per caarlos0/env.
type Config struct {
	Addr string `env:"MCP_SERVER_ADDR" envDefault:":8083"`
	Name string `env:"MCP_SERVER_NAME" envDefault:"mocktool"`
}

const serverVersion = "1.0.0"

// Build wires the MCP server from registry + listener address. It is exported
// so tests can build a server in-process.
func Build(reg *tools.Registry, cfg Config) (*mcpsrv.SSEServer, error) {
	if reg == nil {
		return nil, fmt.Errorf("mcpserver: nil registry")
	}
	mcps := mcpsrv.NewMCPServer(
		cfg.Name,
		serverVersion,
		mcpsrv.WithToolCapabilities(false),
		mcpsrv.WithLogging(),
		mcpsrv.WithRecovery(),
	)
	for _, t := range reg.List() {
		registerTool(mcps, reg, t)
	}
	sse := mcpsrv.NewSSEServer(mcps)
	return sse, nil
}

func registerTool(s *mcpsrv.MCPServer, reg *tools.Registry, t tools.Tool) {
	mcpTool := mcplib.NewToolWithRawSchema(t.Name, t.Description, t.InputSchema)

	handler := func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		// mcp-go gives us decoded args; re-encode to raw JSON so the registry
		// handler can decode into its own typed args struct (single owner of
		// arg shape).
		raw, err := json.Marshal(req.GetArguments())
		if err != nil {
			return mcplib.NewToolResultErrorFromErr("encode args", err), nil
		}
		result, err := reg.Invoke(ctx, t.Name, raw)
		if err != nil {
			return mcplib.NewToolResultErrorFromErr(t.Name+" failed", err), nil
		}
		payload, err := json.Marshal(result)
		if err != nil {
			return mcplib.NewToolResultErrorFromErr("encode result", err), nil
		}
		return mcplib.NewToolResultText(string(payload)), nil
	}
	s.AddTool(mcpTool, handler)
}

// Start runs the SSE server on cfg.Addr until ctx is cancelled.
func Start(ctx context.Context, sse *mcpsrv.SSEServer, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("mcpserver listening", "addr", addr)
		if err := sse.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sse.Shutdown(shutdownCtx); err != nil {
			slog.Warn("mcpserver shutdown error", "error", err)
		}
		return ctx.Err()
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
		return nil
	}
}

// Module is the fx wiring for the mcpserver. It expects the global
// *configs.Config plus a fully-populated tools.Deps to be available.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(loadConfig),
		fx.Provide(tools.BuildAll),
		fx.Invoke(register),
	)
}

func loadConfig() Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		panic(err)
	}
	return c
}

func register(lc fx.Lifecycle, reg *tools.Registry, cfg Config) error {
	sse, err := Build(reg, cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := Start(ctx, sse, cfg.Addr); err != nil && err != context.Canceled {
					slog.Error("mcpserver exited", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return nil
}
