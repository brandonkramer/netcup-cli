package cmd

import (
	"github.com/brandonkramer/netcup-cli/internal/mcpserver"
	"github.com/brandonkramer/netcup-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the netcup MCP server on stdio (for Codex / Cursor / Claude)",
		Long: `Start the Model Context Protocol server on stdin/stdout.

Thin tools shell out to this same binary with --format json. Prefer curated
tools; use netcup_call / netcup_cli for full CLI coverage. Do not invoke from a
TTY for interactive use — hosts spawn this as a subprocess.

Example MCP config:
  command: netcup
  args: ["mcp"]`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Avoid JSON envelopes on stdout — MCP owns the stream.
			if err := mcpserver.Serve(version); err != nil {
				return output.Exit(output.ExitAPI, "mcp server: "+err.Error())
			}
			return nil
		},
	}
}
