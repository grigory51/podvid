package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/mcpserver"
)

func init() {
	rootCmd.AddCommand(mcpCmd())
}

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server exposing podvid operations to agents",
		Long: "Starts a Model Context Protocol server over stdio. Connect it from an " +
			"MCP client (e.g. Claude) to manage podcasts and episodes the same way " +
			"you would in the TUI or CLI. Destructive operations are not exposed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			return mcpserver.Run(context.Background(), cfg)
		},
	}
}
