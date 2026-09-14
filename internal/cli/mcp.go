package cli

import (
	"context"

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

type pingOut struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func mcpServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "dsmctl", Version: Version}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "dsm_ping", Description: "Health check: returns the server name and version."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, pingOut, error) {
			return nil, pingOut{Name: "dsmctl", Version: Version}, nil
		})
	return s
}
