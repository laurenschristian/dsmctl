package cli

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/laurenschristian/dsmctl/internal/dsm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP server over stdio (for Claude, Cursor, etc.)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpServer().Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}
}

// rawOut wraps a raw payload so the MCP schema stays a generic object, not an
// array-of-integer (which a bare json.RawMessage becomes).
type rawOut struct {
	Data any `json:"data"`
}

func rawWrap(b json.RawMessage) rawOut {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return rawOut{Data: string(b)}
	}
	return rawOut{Data: v}
}

type emptyIn struct{}

func mcpServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "dsmctl", Version: Version}, nil)

	add := func(name, desc string, fn func(ctx context.Context) (rawOut, error)) {
		mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
			func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, rawOut, error) {
				var out rawOut
				err := authed(ctx, func() (e error) {
					out, e = fn(ctx)
					return
				})
				return nil, out, err
			})
	}

	mcp.AddTool(s, &mcp.Tool{Name: "dsm_status", Description: "Model, DSM version, serial, CPU, RAM, uptime, temperature."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, *dsm.SystemInfo, error) {
			var out *dsm.SystemInfo
			err := authed(ctx, func() (e error) {
				out, e = client.System(ctx)
				return
			})
			return nil, out, err
		})

	add("dsm_health", "CPU and memory load plus volume and disk SMART summary.", func(ctx context.Context) (rawOut, error) {
		u, err := client.Utilization(ctx)
		if err != nil {
			return rawOut{}, err
		}
		st, err := client.Storage(ctx)
		if err != nil {
			return rawOut{}, err
		}
		return rawOut{Data: map[string]any{"utilization": u, "storage": st}}, nil
	})
	add("dsm_storage", "Volumes and physical disks with SMART status.", func(ctx context.Context) (rawOut, error) {
		st, err := client.Storage(ctx)
		if err != nil {
			return rawOut{}, err
		}
		return rawOut{Data: st}, nil
	})
	add("dsm_packages", "Installed packages and versions.", func(ctx context.Context) (rawOut, error) {
		b, err := client.Packages(ctx)
		return rawWrap(b), err
	})
	add("dsm_docker", "Docker containers and their state.", func(ctx context.Context) (rawOut, error) {
		b, err := client.DockerContainers(ctx)
		return rawWrap(b), err
	})
	add("dsm_tasks", "Scheduled tasks.", func(ctx context.Context) (rawOut, error) {
		b, err := client.Tasks(ctx)
		return rawWrap(b), err
	})
	add("dsm_shares", "Shared folders.", func(ctx context.Context) (rawOut, error) {
		b, err := client.Shares(ctx)
		return rawWrap(b), err
	})
	add("dsm_users", "Local users.", func(ctx context.Context) (rawOut, error) {
		b, err := client.Users(ctx)
		return rawWrap(b), err
	})
	add("dsm_connections", "Active service connections (who is connected).", func(ctx context.Context) (rawOut, error) {
		b, err := client.Connections(ctx)
		return rawWrap(b), err
	})

	mcp.AddTool(s, &mcp.Tool{Name: "dsm_raw", Description: "Call any DSM API. api like SYNO.Core.System, method like info, version int, params key=value list."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in rawIn) (*mcp.CallToolResult, rawOut, error) {
			p := url.Values{}
			for _, kv := range in.Params {
				if k, v, ok := strings.Cut(kv, "="); ok {
					p.Set(k, v)
				}
			}
			version := in.Version
			if version == 0 {
				version = 1
			}
			var out rawOut
			err := authed(ctx, func() error {
				b, e := client.Raw(ctx, in.API, in.Method, version, p)
				out = rawWrap(b)
				return e
			})
			return nil, out, err
		})

	return s
}

type rawIn struct {
	API     string   `json:"api" jsonschema:"the DSM API, e.g. SYNO.Core.System"`
	Method  string   `json:"method" jsonschema:"the API method, e.g. info"`
	Version int      `json:"version,omitempty" jsonschema:"API version, default 1"`
	Params  []string `json:"params,omitempty" jsonschema:"extra key=value params"`
}
